package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/profile"
)

var (
	log = golog.Logger("trigger")

	// ErrUnexpectedType indicates the trigger type is unexpected
	ErrUnexpectedType = fmt.Errorf("unexpected trigger type")
	// ErrTypeMismatch indicates the given TriggerType does not match the expected TriggerType
	ErrTypeMismatch = fmt.Errorf("TriggerType mismatch")
	// ErrEmptyOwnerID indicates the given Source has an empty ScopeID, known in other systems as the OwnerID
	ErrEmptyOwnerID = fmt.Errorf("empty OwnerID")
	// ErrEmptyWorkflowID indicates the given Source has an empty WorkflowID
	ErrEmptyWorkflowID = fmt.Errorf("empty WorkflowID")
	// ErrNotFound indicates that the trigger cannot be found
	ErrNotFound = fmt.Errorf("trigger not found")
)

const charset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// NewID returns a random string ID of alphanumeric characters
// These IDs only have to be unique within a single workflow
// This can be replaced with a determinate `NewID` function for testing
var NewID = func() string {
	b := make([]byte, 5)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// A Trigger determines under what circumstances an `event.ETAutomationWorkflowTrigger`
// should be emitted on the given event.Bus. It knows how to `Advance` itself.
type Trigger interface {
	json.Marshaler
	json.Unmarshaler
	// ID returns the Trigger ID
	ID() string
	// Active returns whether the Trigger is enabled
	Active() bool
	// SetActive sets the enabled status
	SetActive(active bool) error
	// Type returns the Type of this Trigger
	Type() string
	// Advance adjusts the Trigger once it has been triggered
	Advance() error
	// ToMap returns the trigger as a map[string]interface
	ToMap() map[string]interface{}
}

// A Listener emits a `event.ETTriggerWorkflow` event when a specific stimulus
// is triggered
type Listener interface {
	// ConstructTrigger returns a Trigger of the associated Type
	ConstructTrigger(opts map[string]interface{}) (Trigger, error)
	// Listen takes a list of sources and adds or updates the Listener's
	// store to include all the active triggers of the correct type
	Listen(source ...Source) error
	// Type returns the type of Trigger that this Listener listens for
	Type() string
	// Start puts the Listener in an active state of listening for triggers
	Start(ctx context.Context) error
	// Stop stops the Listener from listening for triggers
	Stop() error
}

// Source is an abstraction for a `workflow.Workflow`
type Source interface {
	WorkflowID() string
	ActiveTriggers(triggerType string) []map[string]interface{}
	Owner() profile.ID
}
