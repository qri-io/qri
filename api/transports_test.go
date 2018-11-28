package api

import (
	"net/http"
	"testing"
)

func TestHTTPSRedirect(t *testing.T) {
	HTTPSRedirect("foobar")

	go HTTPSRedirect(":5432")

	cli := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.String() != "https://localhost:5432/foo" {
				t.Errorf("expected to be redirect to https. got: %s", req.URL.String())
			}
			if req.Response.StatusCode != http.StatusMovedPermanently {
				t.Errorf("expected permanent redirect. got: %d", req.Response.StatusCode)
			}
			return nil
		},
	}

	req, _ := http.NewRequest("GET", "http://localhost:5432/foo", nil)
	cli.Do(req)
}
