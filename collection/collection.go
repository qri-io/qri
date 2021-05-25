// Package collection maintains a list of user datasets
package collection

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/params"
	"github.com/qri-io/qri/profile"
)

// RootUserKeyID is a special key identifer that lists *all* datasets in a
// collection
const RootUserKeyID = profile.ID("root")

// Collection is a set of datasets scoped to a user profile. A users's
// collection may consist of datasets they have created and datasets added from
// other users. Collections are the canonical source of truth for listing a
// users datasets in a qri instance. While a collection owns the list, the
// fields in a collection item are cached values gathered from other subsystems,
// and must be kept in sync as subsystems mutate their state.
type Collection interface {
	List(ctx context.Context, pid profile.ID, lp params.List) ([]Item, error)
}

// Writable is an extension interface for collection that adds methods for
// adding and removing items
type Writable interface {
	Collection
	Put(ctx context.Context, profileID profile.ID, items ...Item) error
	Delete(ctx context.Context, profileID profile.ID, initIDs ...string) error
}

// Item is a cache of values gathered from other subsystems
// it's a slightly-modified version of the flatbuffer defined in dscache.fbs
type Item struct {
	ProfileID profile.ID // profileID for the author of the dataset
	InitID    string     // init-id derived from logbook, never changes for the same dataset

	TopIndex int // point to logbook entry for newest commit for this dataset
	// CursorIndex int // point to logbook entry for data that is currently in use
	Username string
	Name     string // human readable name for a dataset, can be changed over time

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

	// Automation fields
	WorkflowID     string
	WorkflowStatus string
	// info about applied transform script during ref creation
	RunID string // either Commit.RunID, or the ID of a failed run when no path value (version is present)
	// RunStatus string     // RunStatus is a string version of the run.Status enumeration eg "running", "failed"
	RunDuration string // duration of run execution in nanoseconds
}

type collection struct {
	items map[profile.ID][]Item
}

var (
	_ Collection = (*collection)(nil)
	_ Writable   = (*collection)(nil)
)

// NewLocalCollection constructs a node-local collection
func NewLocalCollection(ctx context.Context, bus event.Bus) Collection {
	return &collection{
		items: make(map[profile.ID][]Item),
	}
}

func (c collection) List(ctx context.Context, pid profile.ID, lp params.List) ([]Item, error) {
	return nil, fmt.Errorf("not finished")
}

func (c *collection) Put(ctx context.Context, pid profile.ID, items ...Item) error {
	for _, item := range items {
		if err := c.putOne(pid, item); err != nil {
			return err
		}
	}
	return nil
}

func (c *collection) putOne(pid profile.ID, item Item) error {
	if item.ProfileID == "" {
		return fmt.Errorf("profileID is required")
	}
	if item.InitID == "" {
		return fmt.Errorf("initID is required")
	}
	if item.Username == "" {
		return fmt.Errorf("username is required")
	}
	if item.Name == "" {
		return fmt.Errorf("name is required")
	}

	c.items[pid] = append(c.items[pid], item)
	return nil
}

func (c collection) Delete(ctx context.Context, pid profile.ID, ids ...string) error {
	return fmt.Errorf("not finished")
}
