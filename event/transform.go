package event

const (
	// ETPrint is a debug-level message
	ETDebug = Type("transform:Debug")
	// ETPrint is a generic info event
	ETPrint = Type("transform:Print")
	// ETWarn is a warning message
	ETWarn = Type("transform:Warn")
	// ETError is an error message
	ETError = Type("transform:Error")
	// ETReference is a dataset reference within a transform
	ETReference = Type("transform:Reference")
	// ETDataset is a dataset document in a transform
	ETDataset = Type("transform:Dataset")
	// ETChangeReport is a change report in a transform
	ETChangeReport = Type("transform:ChangeReport")
	// ETHistory is a dataset history within a transform
	ETHistory = Type("transform:History")
	// ETProfile is a user profile within a transform
	ETProfile = Type("transform:Profile")

	// ETVersionSaved indicates version creation in a transform
	ETVersionSaved = Type("transform:VersionSaved")
	// ETVersionPulled marks version pulling in a transform
	ETVersionPulled = Type("transform:VersionPulled")
	// ETVersionPushed indicates a version
	ETVersionPushed = Type("transform:VersionPushed")
	// ETHistoryChanged indicates an update to logbook within a transform
	ETHistoryChanged = Type("transform:HistoryChanged")

	// ETTransformStart marks the begnning of a transform execution
	ETTransformStart = Type("transform:TransformStart")
	// ETTransformStop marks the end of transform execuction
	ETTransformStop = Type("transform:TransformStop")
	// ETTransformSkip is a combined start/stop indicator showing a transform did
	// not execute
	ETTransformSkip = Type("transform:TransformSkip")

	// ETTransformStepStart marks the start of a transform step execution
	ETTransformStepStart = Type("transform:TransformStepStart")
	// ETTransformStepStop marks the end of a transform step execution
	ETTransformStepStop = Type("transform:TransformStepStop")
	// ETTransformStepSkip is a combined start/stop indicator showing a step did
	// not execute
	ETTransformStepSkip = Type("transform:TransformStepSkip")

	// ETHttpRequestStart marks the start of an HTTP request
	ETHttpRequestStart = Type("transform:HttpRequestStart")
	// ETHttpRequestStop marks the completion of an HTTP request
	ETHttpRequestStop = Type("transform:HttpRequestStop")
)

// TransformMessage is the payload of an ETPrint event
type TransformMessage struct {
	Msg string `json:"msg"`
}

// TransformLifecycle captures state about the execution of a transform script
// it's the payload of ETTransformStart/Stop/Skip
type TransformLifecycle struct {
	RunID  string `json:"runID"`
	Status string `json:"status,omitempty"`
}

// TransformStepLifecycle captures state about the execution of a transform
// script. Payload of ETTransformStepStart/Stop/Skip
type TransformStepLifecycle struct {
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
	Msg    string `json:"msg,omitempty"`
	Error  string `json:"error,omitempty"`
}

// HTTPRequestDetails describes an HTTP request within a transform script
// payload of ETHTTPRequestStart/Stop events
type HTTPRequestDetails struct {
	ID           string `json:"id,omitempty"`
	UploadSize   int    `json:"uploadSize,omitempty"`
	DownloadSize int    `json:"downloadSize,omitempty"`
	Method       string `json:"method,omitempty"`
	URL          string `json:"url,omitempty"`
}
