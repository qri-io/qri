package config

import (
	"reflect"

	"github.com/qri-io/jsonschema"
)

// Filesystem configures the filesystem for qri content
type Filesystem struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
	Source string                 `json:"source,omitempty"`
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (cfg *Filesystem) SetArbitrary(key string, val interface{}) error {
	return nil
}

// DefaultFilesystemIPFS returns a new default ipfs filesystem configuration
func DefaultFilesystemIPFS() *Filesystem {
	return &Filesystem{
		Type: "ipfs",
		Config: map[string]interface{}{
			"path":   "./ipfs",
			"api":    true,
			"pubsub": false,
		},
		Source: "",
	}
}

// DefaultFilesystemHTTP returns a new default ipfs filesystem configuration
func DefaultFilesystemHTTP() *Filesystem {
	return &Filesystem{
		Type: "http",
	}
}

// DefaultFilesystemLocal returns a new default ipfs filesystem configuration
func DefaultFilesystemLocal() *Filesystem {
	return &Filesystem{
		Type: "local",
	}
}

// Validate validates all fields of filesystem returning all errors found.
func (cfg Filesystem) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "Filesystem",
    "description": "Config for the qri content filesystem",
    "type": "object",
    "required": ["type"],
    "properties": {
    	"type": {
        "description": "Type of filesystem",
        "type": "string",
        "enum": [
			"http",
			"ipfs",
			"local",
			"map",
			"mem"
        ]
      }
    }
  }`)
	return validate(schema, &cfg)
}

// Copy returns a deep copy of the Filesystem struct
func (cfg *Filesystem) Copy() *Filesystem {
	res := &Filesystem{
		Type:   cfg.Type,
		Config: cfg.Config,
		Source: cfg.Source,
	}

	return res
}

// Filesystems is a type alias to wrap around an array of Filesystem structs
type Filesystems []*Filesystem

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (cfg *Filesystems) SetArbitrary(key string, val interface{}) error {
	return nil
}

// DefaultFilesystems returns a new default filesystems configuration
func DefaultFilesystems() *Filesystems {
	return &Filesystems{
		DefaultFilesystemIPFS(),
		DefaultFilesystemHTTP(),
		DefaultFilesystemLocal(),
	}
}

// Validate validates all fields of filesystem returning all errors found.
func (cfg Filesystems) Validate() error {
	for _, filesystem := range cfg {
		if !reflect.ValueOf(filesystem).IsNil() {
			if err := filesystem.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

// Copy returns a deep copy of the Filesystem struct
func (cfg *Filesystems) Copy() *Filesystems {
	res := &Filesystems{}
	for _, f := range *cfg {
		*res = append(*res, f.Copy())
	}
	return res
}
