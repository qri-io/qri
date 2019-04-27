package config

import "github.com/qri-io/jsonschema"

// Update configures a Remote Procedure Call (Update) listener
type Update struct {
	Daemonize bool `json:"daemonize"`
	Port      int  `json:"port"`
}

// DefaultUpdatePort is local the port Update serves on by default
var DefaultUpdatePort = 2506

// DefaultUpdate creates a new default Update configuration
func DefaultUpdate() *Update {
	return &Update{
		Daemonize: true,
		Port:      DefaultUpdatePort,
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
    "required": ["daemonize", "port"],
    "properties": {
      "deamonize": {
        "description": "When true, the update service starts as a daemonized process",
        "type": "boolean"
      },
      "port": {
        "description": "port update service will listen for rpc calls",
        "type": "integer"
      }
    }
  }`)
	return validate(schema, &cfg)
}

// Copy makes a deep copy of the Update struct
func (cfg *Update) Copy() *Update {
	res := &Update{
		Daemonize: cfg.Daemonize,
		Port:      cfg.Port,
	}

	return res
}
