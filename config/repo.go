package config

import (
	"fmt"

	"github.com/qri-io/jsonschema"
)

// Repo configures a qri repo
type Repo struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
}

// Ignore implements the ignorer interface from base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct
// and are either deprecaated or no longer supported.
// This simply ignores all defined ignore fields at read time.
func (cfg *Repo) Ignore(key string) error {
	ignoredPaths := map[string]bool{
		"middleware": true,
	}
	if ignoredPaths[key] {
		return nil
	}
	return fmt.Errorf("key '%s' not found", key)
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
