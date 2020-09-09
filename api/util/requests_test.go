package util

import (
	"fmt"
	"net/http"
	"testing"
)

func TestRequestParamInt(t *testing.T) {
	cases := []struct {
		value       string
		expect, def int
	}{
		{"", 0, 0},
		{"", 1, 1},
		{"", -1, -1},
		{"-1", -1, 0},
		{"10", 10, 0},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("def_%d_input_%s", c.def, c.value), func(t *testing.T) {
			r, err := http.NewRequest("GET", "https://example.com", nil)
			if err != nil {
				t.Fatal(err)
			}

			q := r.URL.Query()
			q.Set("key", c.value)
			r.URL.RawQuery = q.Encode()

			got := ReqParamInt(r, "key", c.def)

			if c.expect != got {
				t.Errorf("result mismatch. expected: %d got: %d", c.expect, got)
			}
		})
	}
}

func TestRequestParamBool(t *testing.T) {
	cases := []struct {
		value       string
		expect, def bool
	}{
		{"", false, false},
		{"", true, true},
		{"false", false, true},
		{"true", true, false},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("def_%t_input_%s", c.def, c.value), func(t *testing.T) {
			r, err := http.NewRequest("GET", "https://example.com", nil)
			if err != nil {
				t.Fatal(err)
			}

			q := r.URL.Query()
			q.Set("key", c.value)
			r.URL.RawQuery = q.Encode()

			got := ReqParamBool(r, "key", c.def)

			if c.expect != got {
				t.Errorf("result mismatch. expected: %t got: %t", c.expect, got)
			}
		})
	}
}
