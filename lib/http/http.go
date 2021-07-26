package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	golog "github.com/ipfs/go-log"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	apiutil "github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/auth/token"
)

var log = golog.Logger("lib")

const jsonMimeType = "application/json"

// SourceResolver header name
const SourceResolver = "SourceResolver"

// Client implements the qri http client
type Client struct {
	Address  string
	Protocol string
}

// NewClient instantiates a new Client
func NewClient(multiaddr string) (*Client, error) {
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
	return &Client{
		Address:  httpAddr.String(),
		Protocol: protocol,
	}, nil
}

// NewClientWithProtocol instantiates a new Client with a specified protocol
func NewClientWithProtocol(multiaddr string, protocol string) (*Client, error) {
	maAddr, err := ma.NewMultiaddr(multiaddr)
	if err != nil {
		return nil, err
	}
	httpAddr, err := manet.ToNetAddr(maAddr)
	if err != nil {
		return nil, err
	}
	return &Client{
		Address:  httpAddr.String(),
		Protocol: protocol,
	}, nil
}

// Call calls API endpoint and passes on parameters, context info
func (c Client) Call(ctx context.Context, apiEndpoint string, source string, params interface{}, result interface{}) error {
	return c.CallMethod(ctx, apiEndpoint, http.MethodPost, source, params, result)
}

// CallMethod calls API endpoint and passes on parameters, context info and specific HTTP Method
func (c Client) CallMethod(ctx context.Context, apiEndpoint string, httpMethod string, source string, params interface{}, result interface{}) error {
	// TODO(arqu): work out mimeType configuration/override per API endpoint
	mimeType := jsonMimeType
	addr := fmt.Sprintf("%s://%s%s", c.Protocol, c.Address, apiEndpoint)

	return c.do(ctx, addr, httpMethod, mimeType, source, params, result, false)
}

// CallRaw calls API endpoint and passes on parameters, context info and returns the []byte result
func (c Client) CallRaw(ctx context.Context, apiEndpoint string, source string, params interface{}, result interface{}) error {
	return c.CallMethodRaw(ctx, apiEndpoint, http.MethodPost, source, params, result)
}

// CallMethodRaw calls API endpoint and passes on parameters, context info, specific HTTP Method and returns the []byte result
func (c Client) CallMethodRaw(ctx context.Context, apiEndpoint string, httpMethod string, source string, params interface{}, result interface{}) error {
	// TODO(arqu): work out mimeType configuration/override per API endpoint
	mimeType := jsonMimeType
	addr := fmt.Sprintf("%s://%s%s", c.Protocol, c.Address, apiEndpoint)
	// TODO(arqu): inject context values into headers

	return c.do(ctx, addr, httpMethod, mimeType, source, params, result, true)
}

func (c Client) do(ctx context.Context, addr string, httpMethod string, mimeType string, source string, params interface{}, result interface{}, raw bool) error {
	var req *http.Request
	var err error

	log.Debugf("http: %s - %s", httpMethod, addr)

	if httpMethod == http.MethodGet || httpMethod == http.MethodDelete {
		u, err := url.Parse(addr)
		if err != nil {
			return err
		}

		if params != nil {
			if pm, ok := params.(map[string]string); ok {
				qvars := u.Query()
				for k, v := range pm {
					qvars.Set(k, v)
				}
				u.RawQuery = qvars.Encode()
			}
		}
		req, err = http.NewRequest(httpMethod, u.String(), nil)
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

	if source != "" {
		req.Header.Set(SourceResolver, source)
	}

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
			return fmt.Errorf("Client raw interface is not a byte buffer")
		}
		return nil
	}

	if result != nil {
		resData := apiutil.Response{
			Data: result,
			Meta: &apiutil.Meta{},
		}
		err = json.Unmarshal(body, &resData)
		if err != nil {
			log.Debugf("Client response err: %s", err.Error())
			return fmt.Errorf("Client response err: %s", err)
		}
	}
	return nil
}

func (c Client) checkError(res *http.Response, body []byte, raw bool) error {
	metaResponse := struct {
		Meta *apiutil.Meta
	}{
		Meta: &apiutil.Meta{},
	}
	parseErr := json.Unmarshal(body, &metaResponse)

	if !raw {
		if parseErr != nil {
			log.Debugf("Client response error: %d - %q", res.StatusCode, body)
			return fmt.Errorf("failed parsing response: %q", string(body))
		}
		if metaResponse.Meta == nil {
			log.Debugf("Client response error: %d - %q", res.StatusCode, body)
			return fmt.Errorf("invalid meta response")
		}
	} else if (metaResponse.Meta.Code < 200 || metaResponse.Meta.Code > 299) && metaResponse.Meta.Code != 0 {
		log.Debugf("Client response meta error: %d - %q", metaResponse.Meta.Code, metaResponse.Meta.Error)
		return fmt.Errorf(metaResponse.Meta.Error)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		log.Debugf("Client response error: %d - %q", res.StatusCode, body)
		return fmt.Errorf(string(body))
	}
	return nil
}
