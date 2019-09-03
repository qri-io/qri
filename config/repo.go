package config

import (
	"reflect"

	"github.com/qri-io/jsonschema"
)

// Repo configures a qri repo
type Repo struct {
	Middleware []string `json:"middleware"`
	Type       string   `json:"type"`
	Path       string   `json:"path,omitempty"`
}

// DefaultRepo creates & returns a new default repo configuration
func DefaultRepo() *Repo {
	return &Repo{
		Type:       "fs",
		Middleware: []string{},
	}
}

// Validate validates all fields of repo returning all errors found.
func (cfg Repo) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "Repo",
    "description": "Config for the qri repository",
    "type": "object",
    "required": ["type"],
    "properties": {
      "middleware": {
        "description": "Middleware packages that need to be applied to the repo",
        "type": ["array", "null"],
        "items": {
          "type": "string"
        }
      },
      "type": {
        "description": "Type of repository",
        "type": "string",
        "enum": [
          "fs",
          "mem"
        ]
      }
    }
  }`)
	return validate(schema, &cfg)
}

// Copy returns a deep copy of the Repo struct
func (cfg *Repo) Copy() *Repo {
	res := &Repo{
		Type: cfg.Type,
	}
	if cfg.Middleware != nil {
		res.Middleware = make([]string, len(cfg.Middleware))
		reflect.Copy(reflect.ValueOf(res.Middleware), reflect.ValueOf(cfg.Middleware))
	}

	return res
}
