package event

import (
	"time"
)

const (
	// ETFSICreateLinkEvent type for when FSI creates a link between a dataset
	// and working directory
	ETFSICreateLinkEvent = Topic("fsi:CreateLinkEvent")
)

// FSICreateLinkEvent describes an FSI created link
type FSICreateLinkEvent struct {
	FSIPath  string `json:"fsiPath"`
	Username string `json:"username"`
	Dsname   string `json:"dsName"`
}

const (
	// ETCreatedNewFile is the event for creating a new file
	ETCreatedNewFile = Topic("watchfs:CreatedNewFile")
	// ETModifiedFile is the event for modifying a file
	ETModifiedFile = Topic("watchfs:ModifiedFile")
	// ETDeletedFile is the event for deleting a file
	ETDeletedFile = Topic("watchfs:DeletedFile")
	// ETRenamedFolder is the event for renaming a folder
	ETRenamedFolder = Topic("watchfs:RenamedFolder")
	// ETRemovedFolder is the event for removing a folder
	ETRemovedFolder = Topic("watchfs:RemovedFolder")
)

// WatchfsChange represents events for filesystem changes
type WatchfsChange struct {
	Username    string    `json:"username"`
	Dsname      string    `json:"dsName"`
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	Time        time.Time `json:"time"`
}
