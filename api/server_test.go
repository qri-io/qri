package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerRoutes(t *testing.T) {
	cases := []struct {
		method, endpoint string
		body             []byte
		resStatus        int
	}{
		// {"GET", "/", nil, 200},
		{"GET", "/datasets", nil, 200},
		{"GET", "/status", nil, 200},
		// {"GET", "/ipfs", nil, 200},
		// {"GET", "/datasets", nil, 200},
		// {"GET", "/datasets", nil, 200},
		// {"GET", "/data", nil, 200},
		// {"GET", "/download", nil, 200},
		// {"GET", "/run", nil, 200},
	}

	client := &http.Client{}

	s, err := New(func(opt *Config) {
		opt.Online = false
		opt.MemOnly = true
	})
	if err != nil {
		t.Error(err.Error())
		return
	}

	server := httptest.NewServer(NewServerRoutes(s))

	for i, c := range cases {
		req, err := http.NewRequest(c.method, server.URL+c.endpoint, bytes.NewReader(c.body))
		if err != nil {
			t.Errorf("case %d error creating request: %s", i, err.Error())
			continue
		}

		res, err := client.Do(req)
		if err != nil {
			t.Errorf("case %d error performing request: %s", i, err.Error())
			continue
		}

		if res.StatusCode != c.resStatus {
			t.Errorf("case %d: %s - %s status code mismatch. expected: %d, got: %d", i, c.method, c.endpoint, c.resStatus, res.StatusCode)
			continue
		}

		env := &struct {
			Meta       map[string]interface{}
			Data       interface{}
			Pagination map[string]interface{}
		}{}

		if err := json.NewDecoder(res.Body).Decode(env); err != nil {
			t.Errorf("case %d: %s - %s error unmarshaling json envelope: %s", i, c.method, c.endpoint, err.Error())
			continue
		}

		if env.Meta == nil {
			t.Errorf("case %d: %s - %s doesn't have a meta field", i, c.method, c.endpoint)
			continue
		}
		if env.Data == nil {
			t.Errorf("case %d: %s - %s doesn't have a data field", i, c.method, c.endpoint)
			continue
		}
	}
}
