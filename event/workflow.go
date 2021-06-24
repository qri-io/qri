package event

const (
	// ETWorkflowTrigger signals that a workflow has been triggered
	// Payload will be a string representation of the `workflow.ID`
	// of the workflow that should be run
	// This event should not block
	ETWorkflowTrigger = Type("workflow:trigger")
)
