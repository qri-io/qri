package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/schema"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	apiutil "github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/auth/token"
)

// ErrUnsupportedRPC is an error for when running a method that is not supported via HTTP RPC
var ErrUnsupportedRPC = errors.New("Warning: method is not suported over RPC")

const jsonMimeType = "application/json"

// decoder maps HTTP requests to input structs
var decoder = schema.NewDecoder()

func init() {
	// TODO(arqu): once APIs have a strict mapping to Params this line
	// should be removed and should error out on unknown keys
	decoder.IgnoreUnknownKeys(true)
}

// HTTPClient implements the qri http client
type HTTPClient struct {
	Address  string
	Protocol string
}

// NewHTTPClient instantiates a new HTTPClient
func NewHTTPClient(multiaddr string) (*HTTPClient, error) {
	maAddr, err := ma.NewMultiaddr(multiaddr)
	if err != nil {
		return nil, err
	}
	// we default to the http protocol
	protocol := "http"
	protocols := maAddr.Protocols()
	httpAddr, err := manet.ToNetAddr(maAddr)
	if err != nil {
		return nil, err
	}
	for _, p := range protocols {
		// if https is present in the multiAddr we preffer that over http
		if p.Code == ma.P_HTTPS {
			protocol = "https"
		}
	}
	return &HTTPClient{
		Address:  httpAddr.String(),
		Protocol: protocol,
	}, nil
}

// NewHTTPClientWithProtocol instantiates a new HTTPClient with a specified protocol
func NewHTTPClientWithProtocol(multiaddr string, protocol string) (*HTTPClient, error) {
	maAddr, err := ma.NewMultiaddr(multiaddr)
	if err != nil {
		return nil, err
	}
	httpAddr, err := manet.ToNetAddr(maAddr)
	if err != nil {
		return nil, err
	}
	return &HTTPClient{
		Address:  httpAddr.String(),
		Protocol: protocol,
	}, nil
}

// Call calls API endpoint and passes on parameters, context info
func (c HTTPClient) Call(ctx context.Context, apiEndpoint APIEndpoint, params interface{}, result interface{}) error {
	return c.CallMethod(ctx, apiEndpoint, http.MethodPost, params, result)
}

// CallMethod calls API endpoint and passes on parameters, context info and specific HTTP Method
func (c HTTPClient) CallMethod(ctx context.Context, apiEndpoint APIEndpoint, httpMethod string, params interface{}, result interface{}) error {
	// TODO(arqu): work out mimeType configuration/override per API endpoint
	mimeType := jsonMimeType
	addr := fmt.Sprintf("%s://%s%s", c.Protocol, c.Address, apiEndpoint)

	return c.do(ctx, addr, httpMethod, mimeType, params, result, false)
}

// CallRaw calls API endpoint and passes on parameters, context info and returns the []byte result
func (c HTTPClient) CallRaw(ctx context.Context, apiEndpoint APIEndpoint, params interface{}, result interface{}) error {
	return c.CallMethodRaw(ctx, apiEndpoint, http.MethodPost, params, result)
}

// CallMethodRaw calls API endpoint and passes on parameters, context info, specific HTTP Method and returns the []byte result
func (c HTTPClient) CallMethodRaw(ctx context.Context, apiEndpoint APIEndpoint, httpMethod string, params interface{}, result interface{}) error {
	// TODO(arqu): work out mimeType configuration/override per API endpoint
	mimeType := jsonMimeType
	addr := fmt.Sprintf("%s://%s%s", c.Protocol, c.Address, apiEndpoint)
	// TODO(arqu): inject context values into headers

	return c.do(ctx, addr, httpMethod, mimeType, params, result, true)
}

func (c HTTPClient) do(ctx context.Context, addr string, httpMethod string, mimeType string, params interface{}, result interface{}, raw bool) error {
	var req *http.Request
	var err error

	if httpMethod == http.MethodGet || httpMethod == http.MethodDelete {
		req, err = http.NewRequest(httpMethod, addr, nil)
	} else if httpMethod == http.MethodPost || httpMethod == http.MethodPut {
		payload, err := json.Marshal(params)
		if err != nil {
			return err
		}
		req, err = http.NewRequest(httpMethod, addr, bytes.NewReader(payload))
	}
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("Accept", mimeType)

	req, added := token.AddContextTokenToRequest(ctx, req)
	if !added {
		log.Debugw("No token was set on an http client request. Unauthenticated requests may fail", "httpMethod", httpMethod, "addr", addr)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err = c.checkError(res, body, raw); err != nil {
		return err
	}

	if raw {
		if buf, ok := result.(*bytes.Buffer); ok {
			buf.Write(body)
		} else {
			return fmt.Errorf("HTTPClient raw interface is not a byte buffer")
		}
		return nil
	}

	resData := apiutil.Response{
		Data: result,
		Meta: &apiutil.Meta{},
	}
	err = json.Unmarshal(body, &resData)
	if err != nil {
		log.Debugf("HTTPClient response err: %s", err.Error())
		return fmt.Errorf("HTTPClient response err: %s", err)
	}
	return nil
}

func (c HTTPClient) checkError(res *http.Response, body []byte, raw bool) error {
	metaResponse := struct {
		Meta *apiutil.Meta
	}{
		Meta: &apiutil.Meta{},
	}
	parseErr := json.Unmarshal(body, &metaResponse)

	if !raw {
		if parseErr != nil {
			log.Debugf("HTTPClient response error: %d - %q", res.StatusCode, body)
			return fmt.Errorf("failed parsing response: %q", string(body))
		}
		if metaResponse.Meta == nil {
			log.Debugf("HTTPClient response error: %d - %q", res.StatusCode, body)
			return fmt.Errorf("invalid meta response")
		}
	} else if (metaResponse.Meta.Code < 200 || metaResponse.Meta.Code > 299) && metaResponse.Meta.Code != 0 {
		log.Debugf("HTTPClient response meta error: %d - %q", metaResponse.Meta.Code, metaResponse.Meta.Error)
		return fmt.Errorf(metaResponse.Meta.Error)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		log.Debugf("HTTPClient response error: %d - %q", res.StatusCode, body)
		return fmt.Errorf(string(body))
	}
	return nil
}

// NewHTTPRequestHandler creates a JSON-API endpoint for a registered dispatch
// method
func NewHTTPRequestHandler(inst *Instance, libMethod string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := inst.NewInputParam(libMethod)
		if p == nil {
			apiutil.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("no params for method %s", libMethod))
			return
		}

		if err := UnmarshalParams(r, p); err != nil {
			log.Debugw("unmarshal request params", "err", err)
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		res, cursor, err := inst.Dispatch(r.Context(), libMethod, p)
		if err != nil {
			apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}

		if cursor != nil {
			apiutil.WritePageResponse(w, res, r, apiutil.PageFromRequest(r))
			return
		}

		apiutil.WriteResponse(w, res)
	}
}

// UnmarshalParams deserialzes a lib req params stuct pointer from an HTTP
// request
//
// Deprecated: This is only exported for one-off handlers in the API package
// can make use of it. Prefer refactoring to use NewHTTPRequestHandler instead.
// once all callers in the api package are removed, unexport this function.
func UnmarshalParams(r *http.Request, p interface{}) error {
	defer func() {
		if defSetter, ok := p.(NZDefaultSetter); ok {
			defSetter.SetNonZeroDefaults()
		}
	}()

	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		if r.Header.Get("Content-Type") == jsonMimeType {
			body, err := snoop(&r.Body)
			if err != nil && err != io.EOF {
				return err
			}
			// this avoids resolving on empty body requests
			// and tries to handle it almost like a GET
			if err != io.EOF {
				if err := json.NewDecoder(body).Decode(p); err != nil {
					return err
				}
			}
		}
	}

	if ru, ok := p.(RequestUnmarshaller); ok {
		return ru.UnmarshalFromRequest(r)
	}

	if err := r.ParseForm(); err != nil {
		return err
	}
	return decoder.Decode(p, r.Form)
}

// snoop reads from an io.ReadCloser and restores it so it can be read again
func snoop(body *io.ReadCloser) (io.ReadCloser, error) {
	if body != nil && *body != nil {
		result, err := ioutil.ReadAll(*body)
		(*body).Close()

		if err != nil {
			return nil, err
		}
		if len(result) == 0 {
			return nil, io.EOF
		}

		*body = ioutil.NopCloser(bytes.NewReader(result))
		return ioutil.NopCloser(bytes.NewReader(result)), nil
	}
	return nil, io.EOF
}
