package config

import (
	"fmt"
	"time"

	"github.com/qri-io/jsonschema"
)

// Remotes is a named set of remote locations
type Remotes map[string]string

// SetArbitrary is for implementing the ArbitrarySetter interface defined by
// base/fill_struct.go
func (r *Remotes) SetArbitrary(key string, val interface{}) (err error) {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("invalid remote value: %s", val)
	}
	(*r)[key] = str
	return nil
}

// Get retrieves an address from the name of remote
func (r *Remotes) Get(name string) (string, bool) {
	if r == nil {
		return "", false
	}
	addr, ok := (*r)[name]
	return addr, ok
}

// Copy creates a copy of a Remotes struct
func (r *Remotes) Copy() *Remotes {
	c := make(map[string]string)
	for k, v := range *r {
		c[k] = v
	}
	return (*Remotes)(&c)
}

// RemoteServer configures Qri to optionally accept requests from clients via
// network calls
type RemoteServer struct {
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
func (cfg *RemoteServer) SetArbitrary(key string, val interface{}) error {
	return nil
}

// Validate validates all fields of render returning all errors found.
func (cfg RemoteServer) Validate() error {
	schema := jsonschema.Must(`{
    "$schema": "http://json-schema.org/draft-06/schema#",
    "title": "RemoteServer",
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

// Copy returns a deep copy of the RemoteServer struct
func (cfg *RemoteServer) Copy() *RemoteServer {
	res := &RemoteServer{
		Enabled:          cfg.Enabled,
		AcceptSizeMax:    cfg.AcceptSizeMax,
		AcceptTimeoutMs:  cfg.AcceptTimeoutMs,
		RequireAllBlocks: cfg.RequireAllBlocks,
		AllowRemoves:     cfg.AllowRemoves,
	}

	return res
}
