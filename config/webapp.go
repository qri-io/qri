package config

import "github.com/qri-io/jsonschema"

// DefaultWebappPort is local the port webapp serves on by default
var DefaultWebappPort = 2505

// Webapp configures the qri webapp service
type Webapp struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
	// token for analytics tracking
	AnalyticsToken string `json:"analyticstoken"`
	// EntrypointHash is a hash of the compiled webapp (the output of running webpack https://github.com/qri-io/frontend)
	// this is fetched and replaced via dnslink when the webapp server starts
	// the value provided here is just a sensible fallback for when dnslink lookup fails,
	// pointing to a known prior version of the the webapp
	EntrypointHash string `json:"entrypointhash"`
}

// DefaultWebapp creates a new default Webapp configuration
func DefaultWebapp() *Webapp {
	return &Webapp{
		Enabled:        true,
		Port:           DefaultWebappPort,
		EntrypointHash: "QmP99mprLUGhMqrh5gyqt4McrgfTJCKCSh5eGJaZw2LycF",
	}
}

// Validate validates all fields of webapp returning all errors found.
func (cfg Webapp) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "Webapp",
    "description": "Config for the webapp",
    "type": "object",
    "required": ["enabled", "port", "analyticstoken", "entrypointhash"],
    "properties": {
      "enabled": {
        "description": "When true, the webapp is accessable from your browser",
        "type": "boolean"
      },
      "port": {
        "description": "The port on which the webapp can be accessed",
        "type": "integer"
      },
      "analyticstoken": {
        "description": "Token for analytics tracking",
        "type": "string"
      },
      "entrypointhash": {
        "description": "A hash of the compiled webapp. This is fetched and replaced via dsnlink when the webapp server starts. The value provided here is just a sensible fallback for when dnslink lookup fails.",
        "type": "string"
      }
    }
  }`)
	return validate(schema, &cfg)
}
