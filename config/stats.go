package config

import (
	"github.com/qri-io/jsonschema"
)

// Stats configures qri statistical metadata calculation
type Stats struct {
	Cache cache `json:"cache"`
	// For later addition:
	// StopFreqCountThreshold int
}

// cache configures the cached storage of stats
type cache struct {
	Type    string `json:"type"`
	MaxSize uint64 `json:"maxsize"`
	Path    string `json:"path,omitempty"`
}

// DefaultStats creates & returns a new default stats configuration
func DefaultStats() *Stats {
	return &Stats{
		Cache: cache{
			Type: "fs",
			// Default to 25MiB
			MaxSize: 1024 * 25,
		},
	}
}

// Validate validates all the fields of stats returning all errors found.
func (cfg Stats) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "Stats",
    "description": "Config for the qri stats cache",
    "type": "object",
    "required": [
      "cache"
    ],
    "properties": {
      "cache": {
        "description": "The configuration for the cache that stores recent calculated stats.",
        "type": "object",
        "required": [
          "type",
          "maxsize"
        ],
        "properties": {
          "maxsize": {
            "description": "The maximum size, in bytes, that the cache should hold",
            "type": "number"
          },
          "path": {
            "description": "The path to the cache. Default is empty. If empty, Qri will save the cache in the Qri Path",
            "type": "string"
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
      }
    }
  }`)
	return validate(schema, &cfg)
}

// Copy returns a deep copy of the Stats struct
func (cfg *Stats) Copy() *Stats {
	return &Stats{
		Cache: cache{
			Type:    cfg.Cache.Type,
			MaxSize: cfg.Cache.MaxSize,
			Path:    cfg.Cache.Path,
		},
	}
}
