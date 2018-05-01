package config

import (
	"fmt"
	"github.com/qri-io/jsonschema"
)

// DefaultAPIPort is local the port webapp serves on by default
var DefaultAPIPort = 2503

// API holds configuration for the qri JSON api
type API struct {
	Enabled bool `json:"enabled"`
	// APIPort specifies the port to listen for JSON API calls
	Port int `json:"port"`
	// read-only mode
	ReadOnly bool `json:"readonly"`
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
	}
}
