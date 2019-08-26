package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/qri/registry"
)

func TestSearch(t *testing.T) {
	reg := registry.Registry{
		Profiles: registry.NewMemProfiles(),
		Search:   &registry.MockSearch{},
	}
	s := httptest.NewServer(NewRoutes(reg))

	cases := []struct {
		method      string
		endpoint    string
		contentType string
		params      *registry.SearchParams
		resStatus   int
	}{
		// TODO (b5) - restore
		// {"GET", "/search", "application/json", &registry.SearchParams{Q: "abc", Limit: 0, Offset: 100}, 400},
	}

	for i, c := range cases {
		req, err := http.NewRequest(c.method, fmt.Sprintf("%s%s", s.URL, c.endpoint), nil)
		if err != nil {
			t.Errorf("case %d error creating request: %s", i, err.Error())
			continue
		}
		if c.contentType != "" {
			req.Header.Set("Content-Type", c.contentType)
		}
		if c.params != nil {
			data, err := json.Marshal(c.params)
			if err != nil {
				t.Errorf("error marshaling json body: %s", err.Error())
				return
			}
			req.Body = ioutil.NopCloser(bytes.NewReader([]byte(data)))
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("case %d unexpected error: %s", i, err)
			continue
		}

		if res.StatusCode != c.resStatus {
			t.Errorf("case %d res status mismatch. expected: %d, got: %d", i, c.resStatus, res.StatusCode)
			continue
		}
	}
}
