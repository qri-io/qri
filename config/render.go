package config

import "github.com/qri-io/jsonschema"

// Render configures the qri render command
type Render struct {
	// TemplateUpdateAddress is currently an IPNS location to check for updates. api.Server starts
	// this address is checked, and if the hash there differs from DefaultTemplateHash, it'll use that instead
	TemplateUpdateAddress string `json:"templateUpdateAddress"`
	// DefaultTemplateHash is a hash of the compiled template
	// this is fetched and replaced via dnslink when the render server starts
	// the value provided here is just a sensible fallback for when dnslink lookup fails,
	// pointing to a known prior version of the the render
	DefaultTemplateHash string `json:"defaultTemplateHash"`
}

// DefaultRender creates a new default Render configuration
func DefaultRender() *Render {
	return &Render{
		TemplateUpdateAddress: "/ipns/defaulttmpl.qri.io",
		DefaultTemplateHash:   "/ipfs/QmeqeRTf2Cvkqdx4xUdWi1nJB2TgCyxmemsL3H4f1eTBaw",
	}
}

// Validate validates all fields of render returning all errors found.
func (cfg Render) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "Render",
    "description": "Render for the render",
    "type": "object",
    "required": ["templateUpdateAddress", "defaultTemplateHash"],
    "properties": {
      "templateUpdateAddress": {
        "description": "address to check for app updates",
        "type": "string"
      },
      "defaultTemplateHash": {
        "description": "A hash of the compiled render. This is fetched and replaced via dsnlink when the render server starts. The value provided here is just a sensible fallback for when dnslink lookup fails.",
        "type": "string"
      }
    }
  }`)
	return validate(schema, &cfg)
}

// Copy returns a deep copy of the Render struct
func (cfg *Render) Copy() *Render {
	res := &Render{
		TemplateUpdateAddress: cfg.TemplateUpdateAddress,
		DefaultTemplateHash:   cfg.DefaultTemplateHash,
	}
	return res
}
