package config

import "github.com/qri-io/jsonschema"

// Update configures a Remote Procedure Call (Update) listener
type Update struct {
	Type      string `json:"type"`
	Daemonize bool   `json:"daemonize"`
	Address   string `json:"address"`
}

// DefaultUpdateAddress is the local address Update serves on by default
var DefaultUpdateAddress = "127.0.0.1:2506"

// DefaultUpdate creates a new default Update configuration
func DefaultUpdate() *Update {
	return &Update{
		Type:      "fs",
		Daemonize: true,
		Address:   DefaultUpdateAddress,
	}
}

// Validate validates all fields of rpc returning all errors found.
func (cfg Update) Validate() error {
	schema := jsonschema.Must(`
    {
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "Update",
    "description": "The Update configuration",
    "type": "object",
    "required": ["type", "daemonize", "address"],
    "properties": {
      "type": {
        "description": "class of cron store",
        "enum": ["mem", "fs"],
        "type": "string"
      },
      "deamonize": {
        "description": "When true, the update service starts as a daemonized process",
        "type": "boolean"
      },
      "address": {
        "description": "address service will listen and dial on for inter-process communication",
        "type": "string"
      }
    }
  }`)
	return validate(schema, &cfg)
}

// Copy makes a deep copy of the Update struct
func (cfg *Update) Copy() *Update {
	res := &Update{
		Type:      cfg.Type,
		Daemonize: cfg.Daemonize,
		Address:   cfg.Address,
	}

	return res
}
