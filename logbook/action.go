package logbook

import (
	"github.com/qri-io/dataset"
)

// ActionType is the type of action that a logbook just completed
type ActionType byte

const (
	// ActionDatasetNameInit is an action that inits a dataset name
	ActionDatasetNameInit ActionType = iota
	// ActionDatasetChange is an action for when a dataset changes
	ActionDatasetChange
)

// Action represents the result of an action that logbook just completed
type Action struct {
	Type       ActionType
	InitID     string
	TopIndex   int
	ProfileID  string
	Username   string
	PrettyName string
	HeadRef    string
	Dataset    *dataset.Dataset
}
