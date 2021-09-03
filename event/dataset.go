package event

const (
	// ETDatasetNameInit occurs when a dataset is first initialized
	// payload is a dsref.VersionInfo
	ETDatasetNameInit = Type("dataset:Init")
	// ETDatasetCommitChange occurs when a dataset's head commit changes
	// payload is a dsref.VersionInfo
	ETDatasetCommitChange = Type("dataset:CommitChange")
	// ETDatasetDeleteAll occurs when a dataset is being deleted
	// payload is an `InitID`
	ETDatasetDeleteAll = Type("dataset:DeleteAll")
	// ETDatasetRename occurs when a dataset gets renamed
	// payload is a dsref.VersionInfo
	ETDatasetRename = Type("dataset:Rename")
	// ETDatasetCreateLink occurs when a dataset gets linked to a working directory
	// payload is a dsref.VersionInfo
	ETDatasetCreateLink = Type("dataset:CreateLink")
	// ETDatasetDownload indicates that a dataset has been downloaded
	// payload is an `InitID` string
	ETDatasetDownload = Type("dataset:Download")

	// ETDatasetSaveStarted occurs when a dataset starts being saved
	// this event is sent asynchronously; the publisher is not blocked
	// payload will be a DsSaveEvent
	ETDatasetSaveStarted = Type("dataset:SaveStarted")
	// ETDatasetSaveProgress occurs whenever a dataset save makes progress
	// this event is sent asynchronously; the publisher is not blocked
	// payload will be a DsSaveEvent
	ETDatasetSaveProgress = Type("dataset:SaveProgress")
	// ETDatasetSaveCompleted occurs when a dataset save finishes
	// payload will be a DsSaveEvent
	ETDatasetSaveCompleted = Type("dataset:SaveCompleted")
)

// DsRename encapsulates fields from a dataset rename
type DsRename struct {
	InitID  string `json:"initID"`
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

// DsSaveEvent represents a change in version creation progress
type DsSaveEvent struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	// either message or error will be populated. message should be human-centric
	// description of progress
	Message string `json:"message"`
	// saving error. only populated on failed ETSaveDatasetCompleted event
	Error error `json:"error,omitempty"`
	// completion pct from 0-1
	Completion float64 `json:"complete"`
	// only populated on successful ETDatasetSaveCompleted
	Path string `json:"path,omitempty"`
}
