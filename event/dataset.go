package event

import (
	"github.com/qri-io/qri/dsref"
)

const (
	// ETDatasetNameInit is when a dataset is initialized
	// payload is a DsChange
	ETDatasetNameInit = Type("dataset:Init")
	// ETDatasetCommitChange is when a dataset changes its newest commit, either
	// when a vesrion is added or some number of versions less then all are
	// removed
	// payload is a DsChange
	ETDatasetCommitChange = Type("dataset:CommitChange")
	// ETDatasetDeleteAll is when a dataset is entirely deleted
	// payload is a DsChange
	ETDatasetDeleteAll = Type("dataset:DeleteAll")
	// ETDatasetRename is when a dataset is renamed
	// payload is a DsChange
	ETDatasetRename = Type("dataset:Rename")
	// ETDatasetCreateLink is when a dataset is linked to a working directory
	// payload is a DsChange
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

// DsChange represents the result of a change to a dataset
type DsChange struct {
	InitID     string             `json:"initID"`
	TopIndex   int                `json:"topIndex"`
	ProfileID  string             `json:"profileID"`
	Username   string             `json:"username"`
	PrettyName string             `json:"prettyName"`
	HeadRef    string             `json:"headRef"`
	Info       *dsref.VersionInfo `json:"info"`
	Dir        string             `json:"dir"`
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
