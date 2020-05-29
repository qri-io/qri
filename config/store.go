package config

import "github.com/qri-io/jsonschema"

// Store configures a qri content addessed file store (cafs)
type Store struct {
	Type    string                 `json:"type"`
	Options map[string]interface{} `json:"options,omitempty"`
	Path    string                 `json:"path,omitempty"`
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (cfg *Store) SetArbitrary(key string, val interface{}) error {
	return nil
}

// DefaultStore returns a new default Store configuration
func DefaultStore() *Store {
	return &Store{
		Type: "ipfs",
		Options: map[string]interface{}{
			"api": true,
		},
	}
}

// Validate validates all fields of store returning all errors found.
func (cfg Store) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "Store",
    "description": "Config for the qri content addressed file store",
    "type": "object",
    "required": ["type"],
    "properties": {
      "type": {
        "description": "Type of store",
        "type": "string",
        "enum": [
					"ipfs",
					"ipfs_http",
					"map"
        ]
      }
    }
  }`)
	return validate(schema, &cfg)
}

// Copy returns a deep copy of the Store struct
func (cfg *Store) Copy() *Store {
	res := &Store{
		Type:    cfg.Type,
		Options: cfg.Options,
		Path:    cfg.Path,
	}

	return res
}
