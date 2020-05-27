package config

import (
	"fmt"
	"reflect"
	"time"

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
	// remote mode
	//
	// Deprecated: use config.Remote instead
	RemoteMode bool `json:"remotemode"`
	// maximum size of dataset to accept for remote mode
	//
	// Deprecated: use config.Remote instead
	RemoteAcceptSizeMax int64 `json:"remoteacceptsizemax"`
	// timeout for remote sessions, in milliseconds
	//
	// Deprecated: use config.Remote instead
	RemoteAcceptTimeoutMs time.Duration `json:"remoteaccepttimeoutms"`
	// URLRoot is the base url for this server
	URLRoot string `json:"urlroot"`
	// TLS enables https via letsEyncrypt
	TLS bool `json:"tls"`
	// Time in seconds to stop the server after,
	// default 0 means keep alive indefinitely
	DisconnectAfter int `json:"disconnectafter,omitempty"`
	// if true, requests that have X-Forwarded-Proto: http will be redirected
	// to their https variant
	ProxyForceHTTPS bool `json:"proxyforcehttps"`
	// support CORS signing from a list of origins
	AllowedOrigins []string `json:"allowedorigins"`
	// whether to allow requests from addresses other than localhost
	ServeRemoteTraffic bool `json:"serveremotetraffic"`
	// APIWebsocketAddress specifies the multiaddress to listen for websocket
	WebsocketAddress string `json:"websocketaddress"`
}

// Validate validates all fields of api returning all errors found.
func (a API) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "api",
    "description": "Config for the api",
    "type": "object",
    "required": ["address", "websocketaddress", "enabled", "readonly", "urlroot", "tls", "proxyforcehttps", "allowedorigins"],
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
      "urlroot": {
        "description": "The base url for this server",
        "type": "string"
      },
      "tls": {
        "description": "Enables https via letsEncrypt",
        "type": "boolean"
      },
      "disconnectafter": {
        "description": "time in seconds to stop the server after",
        "type": "integer"
      },
      "proxyforcehttps": {
        "description": "When true, requests that have X-Forwarded-Proto: http will be redirected to their https variant",
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
		TLS:              false,
		AllowedOrigins: []string{
			"electron://local.qri.io",
			fmt.Sprintf("http://localhost:%d", DefaultWebappPort),
			"http://app.qri.io",
			"https://app.qri.io",
		},
	}
}

// Copy returns a deep copy of an API struct
func (a *API) Copy() *API {
	res := &API{
		Enabled:            a.Enabled,
		Address:            a.Address,
		WebsocketAddress:   a.WebsocketAddress,
		ReadOnly:           a.ReadOnly,
		URLRoot:            a.URLRoot,
		TLS:                a.TLS,
		DisconnectAfter:    a.DisconnectAfter,
		ProxyForceHTTPS:    a.ProxyForceHTTPS,
		ServeRemoteTraffic: a.ServeRemoteTraffic,
	}
	if a.AllowedOrigins != nil {
		res.AllowedOrigins = make([]string, len(a.AllowedOrigins))
		reflect.Copy(reflect.ValueOf(res.AllowedOrigins), reflect.ValueOf(a.AllowedOrigins))
	}
	return res
}
