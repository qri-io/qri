package workflow

import "github.com/qri-io/qri/event"

const (
	// ETWorkflowDeployStarted fires as a scheduler begins adding a workflow to
	// the list of active ("deployed") workflows
	// Payload is a workflow
	// subscriptions do not block the publisher
	ETWorkflowDeployStarted = event.Type("wf:DeployStarted")
	// ETWorkflowDeployStopped signals a workflow deploy process is finished in
	// both success and failure cases
	// Payload is a workflow
	// subscriptions do not block the publisher
	ETWorkflowDeployStopped = event.Type("wf:DeployStopped")
	// ETWorkflowScheduled fires when a workflow is registered for updating, or
	// when a scheduled workflow changes
	// payload is a Workflow
	// subscriptions do not block the publisher
	ETWorkflowScheduled = event.Type("wf:Scheduled")
	// ETWorkflowUnscheduled fires when a workflow is removed from the update
	// schedule payload is a Workflow
	// subscriptions do not block the publisher
	ETWorkflowUnscheduled = event.Type("wf:Unscheduled")
	// ETWorkflowStarted fires when a workflow has started running
	// payload is a Workflow
	// subscriptions do not block the publisher
	ETWorkflowStarted = event.Type("wf:Started")
	// ETWorkflowCompleted fires when a workflow has finished running
	// payload is a Workflow
	// subscriptions do not block the publisher
	ETWorkflowCompleted = event.Type("wf:Completed")
	// ETWorkflowUpdated fires when a workflow has updated
	// its configuration
	ETWorkflowUpdated = event.Type("wf:Updated")
)
