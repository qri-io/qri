package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	apiutil "github.com/qri-io/qri/api/util"
)

const jsonMimeType = "application/json"

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
	// TODO(arqu): inject context values into headers

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

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if raw {
		if res.StatusCode < 200 && res.StatusCode > 299 {
			return fmt.Errorf("HTTPClient req error: %d - %q", res.StatusCode, body)
		}
		result = body
		return nil
	}

	resData := apiutil.Response{
		Data: result,
		Meta: &apiutil.Meta{},
	}
	err = json.Unmarshal(body, &resData)
	if err != nil {
		log.Debugf("HTTPClient unmarshal err: %s", err.Error())
		return err
	}
	return c.checkError(resData.Meta)
}

func (c HTTPClient) checkError(meta *apiutil.Meta) error {
	if meta == nil {
		return fmt.Errorf("HTTPClient req error: invalid meta response")
	}
	if meta.Code != 200 {
		return fmt.Errorf("HTTPClient req error: %d - %q", meta.Code, meta.Error)
	}
	return nil
}
