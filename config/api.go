package config

import (
	"fmt"
	"reflect"

	"github.com/qri-io/jsonschema"
)

var (
	// DefaultAPIPort is the port the webapp serves on by default
	DefaultAPIPort = "2503"
	// DefaultAPIAddress is the multaddr address the webapp serves on by default
	DefaultAPIAddress = fmt.Sprintf("/ip4/127.0.0.1/tcp/%s", DefaultAPIPort)
)

// API holds configuration for the qri JSON api
type API struct {
	// APIAddress specifies the multiaddress to listen for JSON API calls
	Address string `json:"address"`
	// API is enabled
	Enabled bool `json:"enabled"`
	// support CORS signing from a list of origins
	AllowedOrigins []string `json:"allowedorigins"`
	// whether to allow requests from addresses other than localhost
	ServeRemoteTraffic bool `json:"serveremotetraffic"`
	// should the api provide the /webui endpoint?
	EnableWebui bool `json:"enablewebui"`
}

// SetArbitrary is an interface implementation of base/fill/struct in order to
// safely consume config files that have definitions beyond those specified in
// the struct. This simply ignores all additional fields at read time.
func (a *API) SetArbitrary(key string, val interface{}) error {
	return nil
}

// Validate validates all fields of api returning all errors found.
func (a API) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "api",
    "description": "Config for the api",
    "type": "object",
    "required": ["enabled", "address", "allowedorigins", "serveremotetraffic"],
    "properties": {
      "enabled": {
        "description": "When false, the api port does not listen for calls",
        "type": "boolean"
      },
      "address": {
        "description": "The address on which to listen for JSON API calls",
        "type": "string"
      },
      "enablewebui": {
        "description": "when true a /webui endpoint will serve a frontend app",
        "type": "boolean"
      },
      "serveremotetraffic": {
        "description": "whether to allow requests from addresses other than localhost",
        "type": "boolean"
      },
      "allowedorigins": {
        "description": "Support CORS signing from a list of origins",
        "type": "array",
        "items": {
          "type": "string"
        }
      }
    }
  }`)
	return validate(schema, &a)
}

// DefaultAPI returns the default configuration details
func DefaultAPI() *API {
	return &API{
		Enabled: true,
		Address: DefaultAPIAddress,
		AllowedOrigins: []string{
			fmt.Sprintf("http://localhost:%s", DefaultAPIPort),
		},
		EnableWebui: true,
	}
}

// Copy returns a deep copy of an API struct
func (a *API) Copy() *API {
	res := &API{
		Enabled:            a.Enabled,
		Address:            a.Address,
		ServeRemoteTraffic: a.ServeRemoteTraffic,
		EnableWebui:        a.EnableWebui,
	}
	if a.AllowedOrigins != nil {
		res.AllowedOrigins = make([]string, len(a.AllowedOrigins))
		reflect.Copy(reflect.ValueOf(res.AllowedOrigins), reflect.ValueOf(a.AllowedOrigins))
	}
	return res
}
