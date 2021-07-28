package event

const (
	// ETDatasetNameInit is when a dataset is initialized
	// payload is a dsref.VersionInfo
	ETDatasetNameInit = Type("dataset:Init")
	// ETDatasetCommitChange is when a dataset changes its newest commit
	// payload is a dsref.VersionInfo
	ETDatasetCommitChange = Type("dataset:CommitChange")
	// ETDatasetDeleteAll is when a dataset is entirely deleted
	// payload is a dsref.VersionInfo
	ETDatasetDeleteAll = Type("dataset:DeleteAll")
	// ETDatasetRename is when a dataset is renamed
	// payload is a dsref.VersionInfo
	ETDatasetRename = Type("dataset:Rename")
	// ETDatasetCreateLink is when a dataset is linked to a working directory
	// payload is a dsref.VersionInfo
	ETDatasetCreateLink = Type("dataset:CreateLink")

	// ETDatasetSaveStarted fires when saving a dataset starts
	// subscriptions do not block the publisher
	// payload will be a DsSaveEvent
	ETDatasetSaveStarted = Type("dataset:SaveStarted")
	// ETDatasetSaveProgress indicates a change in progress of dataset version
	// creation.
	// subscriptions do not block the publisher
	// payload will be a DsSaveEvent
	ETDatasetSaveProgress = Type("dataset:SaveProgress")
	// ETDatasetSaveCompleted indicates creating a dataset version finished
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
