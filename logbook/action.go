package logbook

import (
	"github.com/qri-io/dataset"
)

// ActionType is the type of action that a logbook just completed
type ActionType byte

const (
	// ActionMoveCursor is an action that moves the cursor for a dataset
	ActionMoveCursor ActionType = iota
)

// Action represents the result of an action that logbook just completed
type Action struct {
	Type     ActionType
	InitID   string
	TopIndex int
	HeadRef  string
	Dataset  *dataset.Dataset
}
