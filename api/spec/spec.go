package spec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AssertHTTPAPISpec runs a test suite of HTTP requests against the given base URL
// to assert it conforms to the qri core API specification. Spec is defined in
// the "open_api_3.yaml" file in the api package
func AssertHTTPAPISpec(t *testing.T, baseURL, specPackagePath string) {
	t.Helper()

	base, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("invalid base url: %s", err)
	}

	testFiles := []string{
		"testdata/working_directory.json",
	}

	for _, path := range testFiles {
		t.Run(filepath.Base(path), func(t *testing.T) {
			ts := mustLoadTestSuite(t, filepath.Join(specPackagePath, path))
			for i, c := range ts {
				if err := c.do(base); err != nil {
					t.Errorf("case %d %s %s:\n%s", i, c.Method, c.Endpoint, err)
				}
			}
		})
	}
}

// TestCase is a single request-response round trip to the API with parameters
// for constructing the request & expectations for assessing the response.
type TestCase struct {
	Endpoint string            // API endpoint to test
	Method   string            // HTTP request method, defaults to "GET"
	Headers  map[string]string // Request HTTP headers
	Body     interface{}       // request body
	Expect   *Response         // Assertions about the response
}

func (c *TestCase) do(u *url.URL) error {
	if c.Method == "" {
		c.Method = http.MethodGet
	}

	u.Path = c.Endpoint

	body, err := c.reqBodyReader()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(c.Method, u.String(), body)
	if err != nil {
		return err
	}
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if exp := c.Expect; exp != nil {
		if exp.Code != 0 && exp.Code != res.StatusCode {
			return fmt.Errorf("response status code mismatch. want: %d got: %d\nresponse body: %s", exp.Code, res.StatusCode, c.resBodyErrString(res))
		}

		for key, expVal := range exp.Headers {
			got := res.Header.Get(key)
			if expVal != got {
				return fmt.Errorf("repsonse header %q mismatch.\nwant: %q\ngot:  %q", key, expVal, got)
			}
		}

	}

	return nil
}

func (c *TestCase) reqBodyReader() (io.Reader, error) {
	switch b := c.Body.(type) {
	case map[string]interface{}:
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(b); err != nil {
			return nil, err
		}
		return buf, nil
	case string:
		return strings.NewReader(b), nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unrecognized type for request body %T", c.Body)
	}
}

func (c *TestCase) decodeResponseBody(res *http.Response) (body interface{}, contentType string, err error) {
	defer res.Body.Close()
	contentType = res.Header.Get("Content-Type")
	switch contentType {
	case "application/json":
		err = json.NewDecoder(res.Body).Decode(&body)
	default:
		body, err = ioutil.ReadAll(res.Body)
	}
	return body, contentType, err
}

func (c *TestCase) resBodyErrString(res *http.Response) string {
	bd, ct, err := c.decodeResponseBody(res)
	if err != nil {
		return err.Error()
	}
	if ct == "application/json" {
		enc, _ := json.MarshalIndent(bd, "", "  ")
		return string(enc)
	}

	if data, ok := bd.([]byte); ok {
		return string(data)
	}

	return fmt.Sprintf("<unexpected response body. Content-Type: %q DataType: %T>", ct, bd)
}

type Response struct {
	Code    int
	Headers map[string]string
}

func mustLoadTestSuite(t *testing.T, filePath string) []*TestCase {
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("opening test suite file %q: %s", filePath, err)
	}
	defer f.Close()
	suite := []*TestCase{}
	if err := json.NewDecoder(f).Decode(&suite); err != nil {
		t.Fatalf("deserializing test suite file %q: %s", filePath, err)
	}

	return suite
}
