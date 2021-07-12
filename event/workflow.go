package event

import (
	"github.com/qri-io/qri/profile"
)

const (
	// ETWorkflowTrigger signals that a workflow has been triggered
	// Payload will be a WorkflowTriggerEvent
	// This event should not block
	ETWorkflowTrigger = Type("workflow:trigger")
	// ETWorkflowStarted signals that a workflow run has begun
	// Payload will be a *WorkflowStartedEvent
	// This event should not block
	ETWorkflowStarted = Type("workflow:started")
	// ETWorkflowStopped signals that a workflow run has finished
	// Payload will be a *WorkflowStoppedEvent
	// This event should not block
	ETWorkflowStopped = Type("workflow:stopped")
	// ETDeployStart signals that a deploy has started
	// Payload will be a DeployEvent
	// This event should not block
	ETDeployStart = Type("deploy:start")
	// ETDeploySaveDatasetStart signals that we have started the save
	// dataset portion of the deploy
	// Payload will be a DeployEvent
	// This event should not block
	ETDeploySaveDatasetStart = Type("deploy:save dataset start")
	// ETDeploySaveDatasetEnd signals that a save dataset has completed as
	// part of a deploy
	// Payload will be a DeployEvent
	// This event should not block
	ETDeploySaveDatasetEnd = Type("deploy:save dataset end")
	// ETDeploySaveWorkflowStart signals the deploy has started the workflow
	// save portion of the deploy
	// Payload will be a DeployEvent
	// This event should not block
	ETDeploySaveWorkflowStart = Type("deploy:save workflow start")
	// ETDeploySaveWorkflowEnd signals the deploy has finished the workflow
	// save portion of the deploy
	// Payload will be a DeployEvent
	// This event should not block
	ETDeploySaveWorkflowEnd = Type("deploy:save workflow end")
	// ETDeployRun signals the deploy has begun the run portion of the
	// deploy
	// Payload will be a DeployEvent
	// This event should not block
	ETDeployRun = Type("deploy:run")
	// ETDeployEnd signals the deploy has finished
	// Payload will be a DeployEvent
	// This event should not block
	ETDeployEnd = Type("deploy:end")
	// ETDeployError signals the deploy has errored
	// Payload will be a DeployEvent
	// This event should not block
	ETDeployError = Type("deploy:error")
)

// WorkflowTriggerEvent is the expected payload of the `ETWorkflowTrigger`
type WorkflowTriggerEvent struct {
	WorkflowID string
	OwnerID    profile.ID
	TriggerID  string
}

// WorkflowStartedEvent is the expected payload of the `ETWorkflowStarted`
type WorkflowStartedEvent struct {
	DatasetID  string
	OwnerID    profile.ID
	WorkflowID string
	RunID      string
}

// WorkflowStoppedEvent is the expected payload of the `ETWorkflowStopped`
type WorkflowStoppedEvent struct {
	DatasetID  string
	OwnerID    profile.ID
	WorkflowID string
	RunID      string
	Status     string
}

// DeployEvent is the expected payload for deploy events
type DeployEvent struct {
	Ref        string
	DatasetID  string
	WorkflowID string
	RunID      string
}

// DeployErrorEvent is the expected payload for deploy error events
type DeployErrorEvent struct {
	DeployEvent
	Error string
}
