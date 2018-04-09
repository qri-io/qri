package config

import "github.com/qri-io/jsonschema"

// RPC configures a Remote Procedure Call (RPC) listener
type RPC struct {
	Enabled bool   `json:"enabled"`
	Port    string `json:"port"`
}

// DefaultRPCPort is local the port RPC serves on by default
var DefaultRPCPort = "2504"

// DefaultRPC creates a new default RPC configuration
func DefaultRPC() *RPC {
	return &RPC{
		Enabled: true,
		Port:    DefaultRPCPort,
	}
}

// Validate validates all fields of rpc returning all errors found.
func (cfg RPC) Validate() error {
	schema := jsonschema.Must(`
    {
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "RPC",
    "description": "The RPC configuration",
    "type": "object",
    "required": ["enabled", "port"],
    "properties": {
      "enabled": {
        "description": "When true, communcation over rpc is allowed",
        "type": "boolean"
      },
      "port": {
        "description": "The port on which to listen for rpc calls",
        "type": "string"
      }
    }
  }`)
	return validate(schema, &cfg)
}
