package event

import (
	"github.com/qri-io/qri/profile"
)

const (
	// ETAutomationWorkflowTrigger signals that a workflow has been triggered
	// Payload will be a WorkflowTriggerEvent
	// This event should not block
	ETAutomationWorkflowTrigger = Type("automation:WorkflowTrigger")
	// ETAutomationWorkflowStarted signals that a workflow run has begun
	// Payload will be a WorkflowStartedEvent
	// This event should not block
	ETAutomationWorkflowStarted = Type("automation:WorkflowStarted")
	// ETAutomationWorkflowStopped signals that a workflow run has finished
	// Payload will be a WorkflowStoppedEvent
	// This event should not block
	ETAutomationWorkflowStopped = Type("automation:WorkflowStopped")
	// ETAutomationDeployStart signals that a deploy has started
	// Payload will be a DeployEvent
	// This event should not block
	ETAutomationDeployStart = Type("automation:DeployStart")
	// ETAutomationDeploySaveDatasetStart signals that we have started the save
	// dataset portion of the deploy
	// Payload will be a DeployEvent
	// This event should not block
	ETAutomationDeploySaveDatasetStart = Type("automation:DeploySaveDatasetStart")
	// ETAutomationDeploySaveDatasetEnd signals that a save dataset has completed as
	// part of a deploy
	// Payload will be a DeployEvent
	// This event should not block
	ETAutomationDeploySaveDatasetEnd = Type("automation:DeploySaveDatasetEnd")
	// ETAutomationDeploySaveWorkflowStart signals the deploy has started the workflow
	// save portion of the deploy
	// Payload will be a DeployEvent
	// This event should not block
	ETAutomationDeploySaveWorkflowStart = Type("automation:DeploySaveWorkflowStart")
	// ETAutomationDeploySaveWorkflowEnd signals the deploy has finished the workflow
	// save portion of the deploy
	// Payload will be a DeployEvent
	// This event should not block
	ETAutomationDeploySaveWorkflowEnd = Type("automation:DeploySaveWorkflowEnd")
	// ETAutomationDeployRun signals the deploy has begun the run portion of the
	// deploy
	// Payload will be a DeployEvent
	// This event should not block
	ETAutomationDeployRun = Type("automation:DeployRun")
	// ETAutomationDeployEnd signals the deploy has finished
	// Payload will be a DeployEvent, if the `Error` field is filled,
	// the deploy ended in error
	ETAutomationDeployEnd = Type("automation:DeployEnd")
)

// WorkflowTriggerEvent is the expected payload of the `ETAutomationWorkflowTrigger`
type WorkflowTriggerEvent struct {
	WorkflowID string     `json:"workflowID"`
	OwnerID    profile.ID `json:"ownerID"`
	TriggerID  string     `json:"triggerID"`
}

// WorkflowStartedEvent is the expected payload of the `ETAutomationWorkflowStarted`
type WorkflowStartedEvent struct {
	DatasetID  string     `json:"datasetID"`
	OwnerID    profile.ID `json:"ownerID"`
	WorkflowID string     `json:"workflowID"`
	RunID      string     `json:"runID"`
}

// WorkflowStoppedEvent is the expected payload of the `ETAutomationWorkflowStopped`
type WorkflowStoppedEvent struct {
	DatasetID  string     `json:"datasetID"`
	OwnerID    profile.ID `json:"ownerID"`
	WorkflowID string     `json:"workflowID"`
	RunID      string     `json:"runID"`
	Status     string     `json:"status"`
}

// DeployEvent is the expected payload for deploy events
type DeployEvent struct {
	Ref        string `json:"ref"`
	DatasetID  string `json:"datasetID"`
	WorkflowID string `json:"workflowID"`
	RunID      string `json:"runID"`
	Error      string `json:"error"`
}
