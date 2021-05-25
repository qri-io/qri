package event

const (
	// ETWorkflowTrigger signals that a workflow has been triggered
	// Payload will be the `workflow.ID` of the workflow that should be run
	// This event should block until the workflow has completed its run
	ETWorkflowTrigger = Type("workflow:trigger")
)
