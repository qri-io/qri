package collection

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/list"
	"github.com/qri-io/qri/profile"
)

// RootUserKeyID is a special key identifer that lists *all* datasets in a
// collection
const RootUserKeyID = profile.ID("root")

type Collection interface {
	List(ctx context.Context, pid profile.ID, lp list.Params) ([]Item, error)
}

// Item is a slightly-modified version of the flatbuffer defined dscache.fbs
type Item struct {
	InitID    string // init-id derived from logbook, never changes for the same dataset
	ProfileID string // profileID for the author of the dataset

	TopIndex    int // point to logbook entry for newest commit for this dataset
	CursorIndex int // point to logbook entry for data that is currently in use
	Username    string
	Name        string // human readable name for a dataset, can be changed over time

	// Meta fields
	MetaTitle string   // metadata title of the dataset
	ThemeList []string // metadata theme of the dataset, comma separated list

	// Structure fields
	BodySize   int    // size of the body in bytes
	BodyRows   int    // number of row in the body
	BodyFormat string // format of the body, such as "csv" or "json"
	NumErrors  int    // number of errors in the structure

	// Commit fields
	CommitTime    time.Time // commit timestamp of the dataset version
	CommitTitle   string    // title field from the commit.
	CommitMessage string

	// About the dataset's history and location
	NumVersions int    // number of versions
	HeadRef     string // the IPFS hash for the dataset
	FSIPath     string // path to checked out working directory for this dataset

	// info about applied transform script during ref creation
	RunID string // either Commit.RunID, or the ID of a failed run when no path value (version is present)
	// RunStatus string     // RunStatus is a string version of the run.Status enumeration eg "running", "failed"
	RunDuration string // duration of run execution in nanoseconds

	WorkflowID     string
	WorkflowStatus string
}

type collection struct {
}

func NewLocalCollection(ctx context.Context, bus event.Bus) Collection {
	return collection{}
}

var _ Collection = (*collection)(nil)

func (c collection) List(ctx context.Context, pid profile.ID, lp list.Params) ([]Item, error) {
	return nil, fmt.Errorf("I'm not finished.")
}
