package config

import (
	"time"

	"github.com/qri-io/jsonschema"
)

// Remote configures Qri for control over the network, accepting dataset push
// requests
type Remote struct {
	// remote mode
	Enabled bool `json:"enabled"`
	// maximum size of dataset to accept for remote mode
	AcceptSizeMax int64 `json:"acceptsizemax"`
	// timeout for remote sessions, in milliseconds
	AcceptTimeoutMs time.Duration `json:"accepttimeoutms"`
	// require clients pushing blocks to send all blocks
	RequireAllBlocks bool `json:"requireallblocks"`
	// allow clients to request unpins for their own pushes
	AllowRemoves bool `json:"allowremoves"`
}

// SetArbitrary is an interface implementation of base/fill/struct in order to safely
// consume config files that have definitions beyond those specified in the struct.
// This simply ignores all additional fields at read time.
func (cfg *Remote) SetArbitrary(key string, val interface{}) error {
	return nil
}

// Validate validates all fields of render returning all errors found.
func (cfg Remote) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "Remote",
    "description": "Configure Qri for control over the network",
    "type": "object",
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

// Copy returns a deep copy of the Remote struct
func (cfg *Remote) Copy() *Remote {
	res := &Remote{
		Enabled:          cfg.Enabled,
		AcceptSizeMax:    cfg.AcceptSizeMax,
		AcceptTimeoutMs:  cfg.AcceptTimeoutMs,
		RequireAllBlocks: cfg.RequireAllBlocks,
		AllowRemoves:     cfg.AllowRemoves,
	}

	return res
}
