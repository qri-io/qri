package feature

import ()

// ID is a string identifier for a workflow
type ID string

// Flag represents a feature and its state
type Flag struct {
	ID     ID   `json:"id"`
	Active bool `json:"active"`
}

var (
	// DefaultFlags represent the base state and system defined feature flags
	DefaultFlags = map[ID]*Flag{
		"BASE": &Flag{
			ID:     "BASE",
			Active: true,
		},
	}
)
