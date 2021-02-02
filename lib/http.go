package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const jsonMimeType = "application/json"

// HTTPClient implements the qri http client
type HTTPClient struct {
	Addr string
}

// NewHTTPClient instantiates a new HTTPClient
func NewHTTPClient(addr string) *HTTPClient {
	return &HTTPClient{
		Addr: addr,
	}
}

// Call resolves the action by mapping it to the correct API endpoint and passes on parameters and context info
func (c HTTPClient) Call(ctx context.Context, action string, params interface{}, result interface{}) error {
	// post should be the default method as we are almost always passing on a call from one instance to another
	httpMethod := http.MethodPost
	mimeType := jsonMimeType
	addr := fmt.Sprintf("http://%s/%%s", c.Addr)

	switch action {
	case "TransformMethods.Apply":
		addr = fmt.Sprintf(addr, "apply")
	case "DatasetMethods.List":
		httpMethod = http.MethodGet
		addr = fmt.Sprintf(addr, "list")
	default:
		return fmt.Errorf("HTTPClient: action not implemented")
	}

	return c.do(ctx, addr, httpMethod, mimeType, params, result)
}

func (c HTTPClient) do(ctx context.Context, addr string, httpMethod string, mimeType string, params interface{}, result interface{}) error {
	var req *http.Request
	var err error

	if httpMethod == "GET" || httpMethod == "DELETE" {
		req, err = http.NewRequest(httpMethod, addr, nil)
	} else if httpMethod == "POST" || httpMethod == "PUT" {
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

	log.Debugf("HTTPClient req.body: %s", body)

	resData := apiResponse{
		Data: result,
		Meta: apiMetaResponse{},
	}
	err = json.Unmarshal(body, &resData)
	if err != nil {
		log.Debugf("HTTPClient unmarshal err: %s", err.Error())
	}
	return c.checkError(resData.Meta)
}

type apiResponse struct {
	Data interface{}     `json:"data"`
	Meta apiMetaResponse `json:"meta"`
}

type apiMetaResponse struct {
	Code   int    `json:"code"`
	Status string `json:"status"`
}

func (c HTTPClient) checkError(meta apiMetaResponse) error {
	if meta.Code != 200 {
		return fmt.Errorf("HTTPClient req error: %d - %q", meta.Code, meta.Status)
	}
	return nil
}
