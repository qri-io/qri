package workflow

import (
// "encoding/json"
)

// TriggerType is the type of trigger
type TriggerType string

// A Trigger determines under what circumstances an `event.ETWorkflowTrigger`
// should be emitted on the given event.Bus. It knows how to `Advance` itself.
type Trigger interface {
	// json.Marshaler
	// json.Unmarshaler
	Enabled() bool
	SetEnable(enabled bool) error
	Type() TriggerType
	Advance() error
	WorkflowID() ID
}
