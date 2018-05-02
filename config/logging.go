package config

import "github.com/qri-io/jsonschema"

// Logging encapsulates logging configuration
type Logging struct {
	// Levels is a map of package_name : log_level (one of [info, error, debug, warn])
	Levels map[string]string `json:"levels"`
}

// DefaultLogging produces a new default logging configuration
func DefaultLogging() *Logging {
	return &Logging{
		Levels: map[string]string{
			"qriapi": "info",
			"qrip2p": "info",
		},
	}
}

// Validate validates all fields of logging returning all errors found.
func (l Logging) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "logging",
    "description": "Config for setting the level of logging output",
    "type": "object",
    "required": ["levels"],
    "properties": {
      "levels": {
        "description": "Levels for logging output for a specific package",
        "type": "object",
        "patternProperties": {
          "^qri": { 
            "type": "string",
            "enum": [
                "info",
                "error",
                "debug",
                "warn"
            ]
          }
        }
      }
    }
  }`)
	return validate(schema, &l)
}

// Copy returns a deep copy of a Logging struct
func (l *Logging) Copy() *Logging {
	res := &Logging{}
	if l.Levels != nil {
		res.Levels = map[string]string{}
		for key, value := range l.Levels {
			res.Levels[key] = value
		}
	}
	return res
}
