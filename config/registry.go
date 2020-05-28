package config

import (
	"github.com/qri-io/jsonschema"
)

// Registry encapsulates configuration options for centralized qri registries
type Registry struct {
	Location string `json:"location"`
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (cfg *Registry) SetArbitrary(key string, val interface{}) error {
	return nil
}

// DefaultRegistry generates a new default registry instance
func DefaultRegistry() *Registry {
	r := &Registry{
		Location: "https://registry.qri.cloud",
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
