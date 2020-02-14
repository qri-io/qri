package watchfs

import (
	"time"
)

// EventType represents the type of event
type EventType string

const (
	// CreateNewFileEvent is the event for creating a new file
	CreateNewFileEvent EventType = "create"
	// ModifyFileEvent is the event for modifying a file
	ModifyFileEvent EventType = "modify"
	// DeleteFileEvent is the event for deleting a file
	DeleteFileEvent EventType = "delete"
	// RenameFolderEvent is the event for renaming a folder
	RenameFolderEvent EventType = "rename"
	// RemoveFolderEvent is the event for removing a folder
	RemoveFolderEvent EventType = "remove"
)

// FilesysEvent represents events for filesystem changes
type FilesysEvent struct {
	Type        EventType
	Username    string
	Dsname      string
	Source      string
	Destination string
	Time        time.Time
}
