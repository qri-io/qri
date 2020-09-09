package config

import (
	"fmt"
	"reflect"

	"github.com/qri-io/jsonschema"
)

// DefaultAPIPort is the port the webapp serves on by default
var DefaultAPIPort = "2503"

// DefaultAPIAddress is the address the webapp serves on by default
var DefaultAPIAddress = fmt.Sprintf("/ip4/127.0.0.1/tcp/%s", DefaultAPIPort)

// DefaultAPIWebsocketAddress is the websocket address the webapp serves on by default
var DefaultAPIWebsocketAddress = "/ip4/127.0.0.1/tcp/2506"

// API holds configuration for the qri JSON api
type API struct {
	// APIAddress specifies the multiaddress to listen for JSON API calls
	Address string `json:"address"`
	// API is enabled
	Enabled bool `json:"enabled"`
	// read-only mode
	ReadOnly bool `json:"readonly"`
	// Time in seconds to stop the server after,
	// default 0 means keep alive indefinitely
	DisconnectAfter int `json:"disconnectafter,omitempty"`
	// support CORS signing from a list of origins
	AllowedOrigins []string `json:"allowedorigins"`
	// whether to allow requests from addresses other than localhost
	ServeRemoteTraffic bool `json:"serveremotetraffic"`
	// WebsocketAddress specifies the multiaddress to listen for websocket
	WebsocketAddress string `json:"websocketaddress"`
	// DisableWebui when true stops qri from serving the webui when the node is online
	// TODO (ramfox): when we next have a config migration, we should probably rename this to
	// EnableWebui and default to true. the double negative here can be confusing.
	DisableWebui bool `json:"disablewebui"`
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
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
    "required": ["address", "websocketaddress", "enabled", "readonly", "allowedorigins"],
    "properties": {
      "enabled": {
        "description": "When false, the api port does not listen for calls",
        "type": "boolean"
      },
      "address": {
        "description": "The address on which to listen for JSON API calls",
        "type": "string"
      },
      "websocketaddress": {
        "description": "The address on which to listen for websocket calls",
        "type": "string"
      },
      "readonly": {
        "description": "When true, api port limits the accepted calls to certain GET requests",
        "type": "boolean"
      },
      "disconnectafter": {
        "description": "time in seconds to stop the server after",
        "type": "integer"
      },
      "disablewebui": {
        "description": "when true, disables qri from serving the webui when the node is online",
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
		Enabled:          true,
		Address:          DefaultAPIAddress,
		WebsocketAddress: DefaultAPIWebsocketAddress,
		AllowedOrigins: []string{
			"electron://local.qri.io",
			"http://app.qri.io",
			"https://app.qri.io",
		},
		DisableWebui: false,
	}
}

// Copy returns a deep copy of an API struct
func (a *API) Copy() *API {
	res := &API{
		Enabled:            a.Enabled,
		Address:            a.Address,
		WebsocketAddress:   a.WebsocketAddress,
		ReadOnly:           a.ReadOnly,
		DisconnectAfter:    a.DisconnectAfter,
		ServeRemoteTraffic: a.ServeRemoteTraffic,
		DisableWebui:       a.DisableWebui,
	}
	if a.AllowedOrigins != nil {
		res.AllowedOrigins = make([]string, len(a.AllowedOrigins))
		reflect.Copy(reflect.ValueOf(res.AllowedOrigins), reflect.ValueOf(a.AllowedOrigins))
	}
	return res
}
