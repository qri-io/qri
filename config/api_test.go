package config

import (
	"testing"
)

func TestAPIValidate(t *testing.T) {
	apiDefault := API{
		Enabled:         true,
		Port:            "2503",
		ReadOnly:        false,
		URLRoot:         "",
		TLS:             false,
		ProxyForceHTTPS: false,
		AllowedOrigins:  []string{"http://localhost:2505"},
	}
	err := apiDefault.Validate()
	if err != nil {
		t.Errorf("error validating apiDefault: %s", err)
	}
}
