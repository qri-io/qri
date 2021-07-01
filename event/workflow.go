package event

import "github.com/qri-io/qri/profile"

const (
	// ETWorkflowTrigger signals that a workflow has been triggered
	// Payload will be a WorkflowTriggerPayload
	// This event should not block
	ETWorkflowTrigger = Type("workflow:trigger")
)

// WorkflowTriggerPayload is the expected payload of the `ETWorkflowTrigger`
type WorkflowTriggerPayload struct {
	WorkflowID string
	OwnerID    profile.ID
	TriggerID  string
}
