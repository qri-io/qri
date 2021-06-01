package trigger

import (
	"encoding/json"
	"fmt"
)

var (
	// ErrUnexpectedType indicates the trigger type is unexpected
	ErrUnexpectedType = fmt.Errorf("unexpected trigger type")
)

// Type is the type of the Trigger
type Type string

// String returns the underlying string associated with the Type
func (tt Type) String() string {
	return string(tt)
}

// A Trigger determines under what circumstances an `event.ETWorkflowTrigger`
// should be emitted on the given event.Bus. It knows how to `Advance` itself.
type Trigger interface {
	json.Marshaler
	json.Unmarshaler
	// Enabled returns whether the Trigger is enabled
	Enabled() bool
	// SetEnabled sets the enabled status
	SetEnabled(enabled bool) error
	// Type returns the Type of this Trigger
	Type() Type
	// Advance adjusts the Trigger once it has been triggered
	Advance() error
}
