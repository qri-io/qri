package config

import (
	"github.com/qri-io/jsonschema"
)

// Stats configures a qri stats
type Stats struct {
	Type    string `json:"type"`
	MaxSize uint64 `json:"maxsize"`
	Path    string `json:"path,omitempty"`
}

// DefaultStats creates & returns a new default stats configuration
func DefaultStats() *Stats {
	return &Stats{
		Type: "fs",
		// Default to 25MiB
		MaxSize: 1024 * 25,
	}
}

// Validate validates all the fields of stats returning all errors found.
func (cfg Stats) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "Stats",
    "description": "Config for the qri stats cache",
    "type": "object",
    "required": ["type","maxsize"],
    "properties": {
      "maxsize": {
        "description": "The maximum size, in bytes, that the cache should hold",
        "type": "number"
      },
      "path": {
        "description": "The path to the cache. Default is empty. If empty, Qri will save the cache in the Qri Path",
        "type":"string"
      },
      "type": {
        "description": "Type of cache",
        "type": "string",
        "enum": [
          "fs",
					"mem",
					"postgres"
        ]
      }
    }
	}`)
	return validate(schema, &cfg)
}

// Copy returns a deep copy of the Stats struct
func (cfg *Stats) Copy() *Stats {
	return &Stats{
		Type:    cfg.Type,
		MaxSize: cfg.MaxSize,
		Path:    cfg.Path,
	}
}
