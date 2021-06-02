package hook

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/qri/event"
)

var (
	// ErrUnexpectedType indicates the hook type is unexpected
	ErrUnexpectedType = fmt.Errorf("unexpected hook type")
)

// Type is the type of hook
type Type string

// String returns the underlying string associated with the Type
func (h Type) String() string { return string(h) }

// A Hook determines under what circumstances its `event.Type` should be
// emitted, and what the event payload should be.
type Hook interface {
	json.Marshaler
	json.Unmarshaler
	// Enabled returns whether the Hook is enabled
	Enabled() bool
	// SetEnabled sets the enabled status
	SetEnabled(enabled bool) error
	// Type returns the Type
	Type() Type
	// Advance adjusts the Hook once it has been triggered
	Advance() error
	// Event returns the event.Type associated with this Hook as well as
	// the payload that should be emitted along with the event
	Event() (event.Type, interface{})
}

type Source interface {
	WorkflowIDString() string
	ActiveHooks(ht Type) []Hook
}
