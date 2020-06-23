package config

import "github.com/qri-io/jsonschema"

// CLI defines configuration details for the qri command line client (CLI)
type CLI struct {
	ColorizeOutput bool `json:"colorizeoutput"`
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (c *CLI) SetArbitrary(key string, val interface{}) error {
	return nil
}

// DefaultCLI returns a new default CLI configuration
func DefaultCLI() *CLI {
	return &CLI{
		ColorizeOutput: true,
	}
}

// Validate validates all fields of cli returning all errors found.
func (c CLI) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "CLI",
    "description": "Config for the CLI",
    "type": "object",
    "required": ["colorizeoutput"],
    "properties": {
      "colorizeoutput": {
        "description": "When true, output to the command line will be colorized",
        "type": "boolean"
      }
    }
  }`)
	return validate(schema, &c)
}

// Copy returns a deep copy of a CLI struct
func (c *CLI) Copy() *CLI {
	res := &CLI{
		ColorizeOutput: c.ColorizeOutput,
	}
	return res
}

// WithPrivateValues returns a deep copy of CLI struct all the privates values of the receiver added to the *CLI param
func (c *CLI) WithPrivateValues(p *CLI) *CLI {
	return p.Copy()
}

// WithoutPrivateValues returns a deep copy of an CLI struct with all the private values removed
func (c *CLI) WithoutPrivateValues() *CLI {
	return c.Copy()
}
