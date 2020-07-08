package event

import (
	"time"
)

const (
	// ETFSICreateLinkEvent type for when FSI creates a link between a dataset
	// and working directory
	ETFSICreateLinkEvent = Type("fsi:CreateLinkEvent")
)

// FSICreateLinkEvent describes an FSI created link
type FSICreateLinkEvent struct {
	FSIPath  string
	Username string
	Dsname   string
}

const (
	// ETCreatedNewFile is the event for creating a new file
	ETCreatedNewFile = Type("watchfs:CreatedNewFile")
	// ETModifiedFile is the event for modifying a file
	ETModifiedFile = Type("watchfs:ModifiedFile")
	// ETDeletedFile is the event for deleting a file
	ETDeletedFile = Type("watchfs:DeletedFile")
	// ETRenamedFolder is the event for renaming a folder
	ETRenamedFolder = Type("watchfs:RenamedFolder")
	// ETRemovedFolder is the event for removing a folder
	ETRemovedFolder = Type("watchfs:RemovedFolder")
)

// WatchfsChange represents events for filesystem changes
type WatchfsChange struct {
	Username    string
	Dsname      string
	Source      string
	Destination string
	Time        time.Time
}
