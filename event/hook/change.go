package hook

import (
	"github.com/qri-io/qri/dsref"
)

// ChangeType is the type of change that has happened to a dataset
type ChangeType byte

const (
	// DatasetNameInit is when a dataset is initialized
	DatasetNameInit ChangeType = iota
	// DatasetCommitChange is when a dataset changes its newest commit
	DatasetCommitChange
	// DatasetDeleteAll is when a dataset is entirely deleted
	DatasetDeleteAll
	// DatasetRename is when a dataset is renamed
	DatasetRename
	// DatasetCreateLink is when a dataset is linked to a working directory
	DatasetCreateLink
)

// DsChange represents the result of a change to a dataset
type DsChange struct {
	Type       ChangeType
	InitID     string
	TopIndex   int
	ProfileID  string
	Username   string
	PrettyName string
	HeadRef    string
	Info       *dsref.VersionInfo
	Dir        string
}

// ChangeNotifier is something that provides a hook which will be called when a dataset changes
type ChangeNotifier interface {
	SetChangeHook(func(change DsChange))
}
