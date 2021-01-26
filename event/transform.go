package event

const (
	// ETTransformStart signals the start a transform execution
	ETTransformStart = Type("transform:Start")
	// ETTransformComplete signals the successful completion of a transform execution
	ETTransformComplete = Type("transform:Complete")
	// ETTransformFailure signals the failure of a transform execution
	ETTransformFailure = Type("transform:Failure")
)
