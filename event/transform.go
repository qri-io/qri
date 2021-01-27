package event

const (
	// ETTransformStart signals the start a transform execution
	ETTransformStart = Type("transform:Start")
	// ETTransformComplete signals the successful completion of a transform execution
	ETTransformComplete = Type("transform:Complete")
	// ETTransformFailure signals the failure of a transform execution
	ETTransformFailure = Type("transform:Failure")

	// ETTransformStepStart signals a step is starting. Payload will be a StepDetail
	ETTransformStepStart = Type("transform:StepStart")
	// ETTransformStepStop signals a step has stopped. Payload will be a StepDetail
	ETTransformStepStop = Type("transform:StepStop")
	// ETTransformStepSkip signals a step was skipped. Payload will be a StepDetail
	ETTransformStepSkip = Type("transform:StepSkip")

	// ETTransformPrint is sent by print commands. Payload will be a Message
	ETTransformPrint = Type("transform:Print")
	// ETTransformError is for when an error occurs. Payload will be a Message
	ETTransformError = Type("transform:Error")
)

// TransformMessage is the payload for print and error events
type TransformMessage struct {
	Msg string `json:"msg"`
}

// TransformStepDetail is the payload for step events
type TransformStepDetail struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Success  bool   `json:"success"`
}
