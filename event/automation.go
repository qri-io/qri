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
	// ETAutomationWorkflowCreated signals that a new workflow has been created
	// Payload will be a workflow.Workflow
	// This event should not block
	ETAutomationWorkflowCreated = Type("automation:WorkflowCreated")
	// ETAutomationWorkflowRemoved signals that a workflow has been removed
	// Payload will be a workflow.Workflow
	// This event should not block
	ETAutomationWorkflowRemoved = Type("automation:WorkflowRemoved")
	// ETAutomationDeployStart signals that a deploy has started
	// Payload will be a DeployEvent
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
	// ETAutomationRunQueuePush signals the run is queued to execute
	// Payload will be a runID
	// This event should not block
	ETAutomationRunQueuePush = Type("automation:RunQueuePush")
	// ETAutomationRunQueuePop signals the run is queued to execute
	// Payload will be a runID
	// This event should not block
	ETAutomationRunQueuePop = Type("automation:RunQueuePop")
	// ETAutomationApplyQueuePush signals the apply is queued to execute
	// Payload will be a runID
	// This event should not block
	ETAutomationApplyQueuePush = Type("automation:ApplyQueuePush")
	// ETAutomationApplyQueuePop signals the apply is queued to execute
	// Payload will be a runID
	// This event should not block
	ETAutomationApplyQueuePop = Type("automation:ApplyQueuePop")
)

// WorkflowTriggerEvent is the expected payload of the `ETAutomationWorkflowTrigger`
type WorkflowTriggerEvent struct {
	WorkflowID string     `json:"workflowID"`
	OwnerID    profile.ID `json:"ownerID"`
	TriggerID  string     `json:"triggerID"`
}

// WorkflowStartedEvent is the expected payload of the `ETAutomationWorkflowStarted`
type WorkflowStartedEvent struct {
	InitID     string     `json:"InitID"`
	OwnerID    profile.ID `json:"ownerID"`
	WorkflowID string     `json:"workflowID"`
	RunID      string     `json:"runID"`
}

// WorkflowStoppedEvent is the expected payload of the `ETAutomationWorkflowStopped`
type WorkflowStoppedEvent struct {
	InitID     string     `json:"InitID"`
	OwnerID    profile.ID `json:"ownerID"`
	WorkflowID string     `json:"workflowID"`
	RunID      string     `json:"runID"`
	Status     string     `json:"status"`
}

// DeployEvent is the expected payload for deploy events
type DeployEvent struct {
	Ref        string `json:"ref"`
	InitID     string `json:"InitID"`
	WorkflowID string `json:"workflowID"`
	RunID      string `json:"runID"`
	Error      string `json:"error"`
}
