package config

import (
	"github.com/qri-io/jsonschema"
)

// Repo configures a qri repo
type Repo struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (cfg *Repo) SetArbitrary(key string, val interface{}) error {
	return nil
}

// DefaultRepo creates & returns a new default repo configuration
func DefaultRepo() *Repo {
	return &Repo{
		Type: "fs",
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

	return res
}
