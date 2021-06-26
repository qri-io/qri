package trigger

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/event"
)

var (
	// ErrTypeMismatch indicates the given TriggerType does not match the expected TriggerType
	ErrTypeMismatch = fmt.Errorf("TriggerType mismatch")
	// ErrEmptyScopeID indicates the given Source has an empty ScopeID, known in other systems as the OwnerID
	ErrEmptyScopeID = fmt.Errorf("empty OwnerID")
	// ErrEmptyWorkflowID indicates the given Source has an empty WorkflowID
	ErrEmptyWorkflowID = fmt.Errorf("empty WorkflowID")
)

// A Listener emits a `event.ETTriggerWorkflow` event when a specific stimulus
// is triggered
type Listener interface {
	// ConstructTrigger returns a Trigger of the associated Type
	ConstructTrigger(opt *Options) (Trigger, error)
	// Listen takes a list of sources and adds or updates the Listener's
	// store to include all the active triggers of the correct type
	Listen(source ...Source) error
	// Type returns the Type of Trigger that this Listener listens for
	Type() Type
	// Start puts the Listener in an active state of listening for triggers
	Start(ctx context.Context) error
	// Stop stops the Listener from listening for triggers
	Stop() error
	// Bus returns the underlying Bus on which the Listener will emit the
	// `event.ETTriggerWorkflow` event
	Bus() event.Bus
}
