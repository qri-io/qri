package config

import (
	"fmt"
	"reflect"
	"time"

	"github.com/qri-io/jsonschema"
)

const (
	// DefaultAPIPort is local the port the HTTP API serves on by default
	DefaultAPIPort = 2503
	// DefaultWebsocketAPIPort is local the port the websocket API serves on by
	// default
	DefaultWebsocketAPIPort = 2506
)

// API holds configuration for the qri JSON api
type API struct {
	Enabled bool `json:"enabled"`
	// the port to listen for JSON API calls
	Port int `json:"port"`
	// read-only mode
	ReadOnly bool `json:"readonly"`
	// remote mode
	//
	// Deprecated: use config.Remote instead
	RemoteMode bool `json:"remotemode,omitempty"`
	// maximum size of dataset to accept for remote mode
	//
	// Deprecated: use config.Remote instead
	RemoteAcceptSizeMax int64 `json:"remoteacceptsizemax,omitempty"`
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

	// websocket flag api.Enabled must be true
	WebsocketEnabled bool `json:"websocketenabled"`
	// port to serve websocket traffic on
	WebsocketPort int `json:"websocketport"`
}

// Validate validates all fields of api returning all errors found.
func (a API) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "api",
    "description": "Config for the api",
    "type": "object",
    "required": ["enabled", "port", "readonly", "urlroot", "tls", "proxyforcehttps", "allowedorigins"],
    "properties": {
      "enabled": {
        "description": "When false, the api port does not listen for calls",
        "type": "boolean"
      },
      "port": {
        "description": "The port that listens for JSON API calls",
        "type": "integer"
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
			},
			"websocketenabled": {
				"description": "Flag to turn on serving qri api over websockets",
				"type": "boolean"
			},
			"websocketport": {
				"description": "local port websocket API should listen on",
				"type": "integer"
			}
    }
  }`)
	return validate(schema, &a)
}

// DefaultAPI returns the default configuration details
func DefaultAPI() *API {
	return &API{
		Enabled: true,
		Port:    DefaultAPIPort,
		TLS:     false,
		AllowedOrigins: []string{
			"electron://local.qri.io",
			fmt.Sprintf("http://localhost:%d", DefaultWebappPort),
			"http://app.qri.io",
			"https://app.qri.io",
		},
		WebsocketEnabled: true,
		WebsocketPort:    DefaultWebsocketAPIPort,
	}
}

// Copy returns a deep copy of an API struct
func (a *API) Copy() *API {
	res := &API{
		Enabled:            a.Enabled,
		Port:               a.Port,
		ReadOnly:           a.ReadOnly,
		URLRoot:            a.URLRoot,
		TLS:                a.TLS,
		DisconnectAfter:    a.DisconnectAfter,
		ProxyForceHTTPS:    a.ProxyForceHTTPS,
		ServeRemoteTraffic: a.ServeRemoteTraffic,
		WebsocketEnabled:   a.WebsocketEnabled,
		WebsocketPort:      a.WebsocketPort,
	}
	if a.AllowedOrigins != nil {
		res.AllowedOrigins = make([]string, len(a.AllowedOrigins))
		reflect.Copy(reflect.ValueOf(res.AllowedOrigins), reflect.ValueOf(a.AllowedOrigins))
	}
	return res
}
