package event

import (
	"github.com/qri-io/qri/profile"
)

const (
	// ETWorkflowTrigger signals that a workflow has been triggered
	// Payload will be a WorkflowTriggerPayload
	// This event should not block
	ETWorkflowTrigger = Type("workflow:trigger")
	// ETWorkflowStarted signals that a workflow run has begun
	// Payload will be a *WorkflowStartedPayload
	// This event should not block
	ETWorkflowStarted = Type("workflow:started")
	// ETWorkflowStopped signals that a workflow run has finished
	// Payload will be a *WorkflowStoppedPayload
	// This event should not block
	ETWorkflowStopped = Type("workflow:stopped")
)

// WorkflowTriggerPayload is the expected payload of the `ETWorkflowTrigger`
type WorkflowTriggerPayload struct {
	WorkflowID string
	OwnerID    profile.ID
	TriggerID  string
}

// WorkflowStartedPayload is the expected payload of the `ETWorkflowStarted`
type WorkflowStartedPayload struct {
	DatasetID  string
	OwnerID    profile.ID
	WorkflowID string
	RunID      string
}

// WorkflowStoppedPayload is the expected payload of the `ETWorkflowStopped`
type WorkflowStoppedPayload struct {
	DatasetID  string
	OwnerID    profile.ID
	WorkflowID string
	RunID      string
	Status     string
}
