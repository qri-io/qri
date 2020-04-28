package logbook

import (
	"github.com/qri-io/qri/dsref"
)

// ActionType is the type of action that a logbook just completed
type ActionType byte

const (
	// ActionDatasetNameInit is an action that inits a dataset name
	ActionDatasetNameInit ActionType = iota
	// ActionDatasetCommitChange is an action for when a dataset changes its newest commit
	ActionDatasetCommitChange
	// ActionDatasetDeleteAll is an action for when a dataset is entirely deleted
	ActionDatasetDeleteAll
	// ActionDatasetRename is when a dataset is renamed
	ActionDatasetRename
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
	Info       *dsref.VersionInfo
}
