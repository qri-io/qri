package event

const (
	// ETTransformStart signals the start a transform execution
	// Payload will be a TransformLifecycle
	ETTransformStart = Type("tf:Start")
	// ETTransformStop signals the completion of a transform execution
	// Payload will be a TransformLifecycle
	ETTransformStop = Type("tf:Stop")

	// ETTransformStepStart signals a step is starting.
	// Payload will be a TransformStepLifecycle
	ETTransformStepStart = Type("tf:StepStart")
	// ETTransformStepStop signals a step has stopped.
	// Payload will be a TransformStepLifecycle
	ETTransformStepStop = Type("tf:StepStop")
	// ETTransformStepSkip signals a step was skipped.
	// Payload will be a TransformStepLifecycle
	ETTransformStepSkip = Type("tf:StepSkip")

	// ETTransformPrint is sent by print commands.
	// Payload will be a Message
	ETTransformPrint = Type("tf:Print")
	// ETTransformError is for when a tranform program execution error occurs.
	// Payload will be a Message
	ETTransformError = Type("tf:Error")
	// ETTransformDatasetPreview is an abbreviated dataset document in a transform
	// Payload will be a *dataset.Dataset Preview
	ETTransformDatasetPreview = Type("tf:DatasetPreview")
)

// TransformLifecycle captures state about the execution of an entire transform
// script
// it's the payload of ETTransformStart/Stop
type TransformLifecycle struct {
	StepCount int    `json:"stepCount"`
	Status    string `json:"status,omitempty"`
	Mode      string `json:"mode,omitempty"`
	InitID    string `json:"initID,omitempty"`
}

// TransformStepLifecycle describes the state of transform step execution at a
// moment in time
// payload for ETTransformStepStart/Stop
type TransformStepLifecycle struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Status   string `json:"status,omitempty"`
	Mode     string `json:"mode,omitempty"`
}

// TransformMsgLvl is an enumeration of all possible degrees of message
// logging in an implicit hierarchy (levels)
type TransformMsgLvl string

const (
	// TransformMsgLvlNone defines an unknown logging level
	TransformMsgLvlNone = TransformMsgLvl("")
	// TransformMsgLvlDebug defines logging level debug
	TransformMsgLvlDebug = TransformMsgLvl("debug")
	// TransformMsgLvlInfo defines logging level info
	TransformMsgLvlInfo = TransformMsgLvl("info")
	// TransformMsgLvlWarn defines logging level warn
	TransformMsgLvlWarn = TransformMsgLvl("warn")
	// TransformMsgLvlError defines logging level error
	TransformMsgLvlError = TransformMsgLvl("error")
)

// TransformMessage is the payload for print and error events
type TransformMessage struct {
	Lvl  TransformMsgLvl `json:"lvl"`
	Msg  string          `json:"msg"`
	Mode string          `json:"mode,omitempty"`
}
