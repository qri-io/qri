package config

import (
	"reflect"
	"testing"
)

func TestAPIValidate(t *testing.T) {
	err := DefaultAPI().Validate()
	if err != nil {
		t.Errorf("error validating default api: %s", err)
	}
}

func TestAPICopy(t *testing.T) {
	cases := []struct {
		description string
		api         *API
	}{
		{"default", DefaultAPI()},
		{"ensure boolean values copy", &API{
			Enabled:            true,
			ReadOnly:           true,
			TLS:                true,
			ProxyForceHTTPS:    true,
			ServeRemoteTraffic: true,
		}},
	}
	for i, c := range cases {
		cpy := c.api.Copy()
		if !reflect.DeepEqual(cpy, c.api) {
			t.Errorf("API Copy test case %d '%s', api structs are not equal: \ncopy: %v, \noriginal: %v", i, c.description, cpy, c.api)
			continue
		}
		if cpy.AllowedOrigins != nil {
			cpy.AllowedOrigins[0] = ""
			if reflect.DeepEqual(cpy, c.api) {
				t.Errorf("API Copy test case %d '%s', editing one api struct should not affect the other: \ncopy: %v, \noriginal: %v", i, c.description, cpy, c.api)
				continue
			}
		}
	}
}
