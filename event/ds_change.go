package event

import (
	"github.com/qri-io/dataset"
)

var (
	// ETDatasetInit is for events that initialize datasets
	ETDatasetInit = Topic("ds:init")
	// ETDatasetChange is for events that change an existing dataset
	ETDatasetChange = Topic("ds:change")
)

// DatasetChangeEvent describes a change to a dataset
type DatasetChangeEvent struct {
	InitID     string
	TopIndex   int
	ProfileID  string
	Username   string
	PrettyName string
	HeadRef    string
	Dataset    *dataset.Dataset
}
