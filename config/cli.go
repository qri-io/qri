package config

import "github.com/qri-io/jsonschema"

// CLI defines configuration details for the qri command line client (CLI)
type CLI struct {
	ColorizeOutput bool `json:"colorizeoutput"`
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
