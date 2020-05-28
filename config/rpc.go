package config

import "github.com/qri-io/jsonschema"

// RPC configures a Remote Procedure Call (RPC) listener
type RPC struct {
	Enabled bool   `json:"enabled"`
	Address string `json:"address"`
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (cfg *RPC) SetArbitrary(key string, val interface{}) error {
	return nil
}

// DefaultRPCAddress is the address RPC serves on by default
var DefaultRPCAddress = "/ip4/127.0.0.1/tcp/2504"

// DefaultRPC creates a new default RPC configuration
func DefaultRPC() *RPC {
	return &RPC{
		Enabled: true,
		Address: DefaultRPCAddress,
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
    "required": ["enabled", "address"],
    "properties": {
      "enabled": {
        "description": "When true, communcation over rpc is allowed",
        "type": "boolean"
      },
      "address": {
        "description": "The address on which to listen for rpc calls",
        "type": "string"
      }
    }
  }`)
	return validate(schema, &cfg)
}

// Copy makes a deep copy of the RPC struct
func (cfg *RPC) Copy() *RPC {
	res := &RPC{
		Enabled: cfg.Enabled,
		Address: cfg.Address,
	}

	return res
}
