package event

const (
	// ETWorkflowTrigger signals that a workflow has been triggered
	// Payload will be the `workflow.ID` of the workflow that should be run
	// This event should block until the workflow has completed its run
	ETWorkflowTrigger = Type("workflow:trigger")
)

const (
	// ETWorkflowDeployStarted fires as a scheduler begins adding a workflow to
	// the list of active ("deployed") workflows
	// Payload is a workflow
	// subscriptions do not block the publisher
	ETWorkflowDeployStarted = Type("wf:DeployStarted")
	// ETWorkflowDeployStopped signals a workflow deploy process is finished in
	// both success and failure cases
	// Payload is a workflow
	// subscriptions do not block the publisher
	ETWorkflowDeployStopped = Type("wf:DeployStopped")
	// ETWorkflowScheduled fires when a workflow is registered for updating, or
	// when a scheduled workflow changes
	// payload is a Workflow
	// subscriptions do not block the publisher
	ETWorkflowScheduled = Type("wf:Scheduled")
	// ETWorkflowUnscheduled fires when a workflow is removed from the update
	// schedule payload is a Workflow
	// subscriptions do not block the publisher
	ETWorkflowUnscheduled = Type("wf:Unscheduled")
	// ETWorkflowStarted fires when a workflow has started running
	// payload is a Workflow
	// subscriptions do not block the publisher
	ETWorkflowStarted = Type("wf:Started")
	// ETWorkflowCompleted fires when a workflow has finished running
	// payload is a Workflow
	// subscriptions do not block the publisher
	ETWorkflowCompleted = Type("wf:Completed")
	// ETWorkflowUpdated fires when a workflow has updated
	// its configuration
	ETWorkflowUpdated = Type("wf:Updated")
)
