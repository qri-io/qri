package config

import (
	"github.com/qri-io/jsonschema"
)

// Registry encapsulates configuration options for centralized qri registries
type Registry struct {
	Location string `json:"location"`
}

// DefaultRegistry generates a new default registry instance
func DefaultRegistry() *Registry {
	r := &Registry{
		Location: "https://registry.qri.io",
	}
	return r
}

// Validate validates all fields of p2p returning all errors found.
func (cfg Registry) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "Registry",
    "description": "Config for registry",
    "type": "object",
    "required": ["location"],
    "properties": {
      "location": {
        "description": "the",
        "type": "string"
      }
    }
  }`)
	return validate(schema, &cfg)
}

// Copy makes a deep copy of the Registry struct
func (cfg *Registry) Copy() *Registry {
	res := &Registry{
		Location: cfg.Location,
	}
	return res
}
