package workflow

import "github.com/qri-io/qri/event"

// "encoding/json"

// HookType is the type of hook
type HookType string

// A Hook determines under what circumstances its `hook.Event()` should be
// emitted, and what the event payload should be.
type Hook interface {
	// json.Marshaler
	// json.Unmarshaler
	Enabled() bool
	SetEnabled(enabled bool) error
	Type() HookType
	Advance() error
	Event() (event.Type, interface{})
}