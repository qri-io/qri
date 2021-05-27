// Package collection maintains a list of user datasets
package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

const collectionsDirName = "collections"

type collection struct {
	basePath string

	sync.Mutex  // collections map lock
	collections map[profile.ID][]Item
}

var (
	_ Collection = (*collection)(nil)
	_ Writable   = (*collection)(nil)
)

// NewLocalCollection constructs a node-local collection, if repoDir is not the
// empty string, localCollection will create a "collections" directory to
// persist collections. providing an empty repoDir value will create an
// in-memory collection
func NewLocalCollection(ctx context.Context, bus event.Bus, repoDir string) (Collection, error) {
	if repoDir == "" {
		// in-memory only collection
		return &collection{
			collections: make(map[profile.ID][]Item),
		}, nil
	}

	repoDir = filepath.Join(repoDir, collectionsDirName)
	fi, err := os.Stat(repoDir)
	if os.IsNotExist(err) {
		if err := os.Mkdir(repoDir, 0755); err != nil {
			return nil, fmt.Errorf("creating collection directory: %w", err)
		}
		return &collection{
			basePath:    repoDir,
			collections: make(map[profile.ID][]Item),
		}, nil
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("collection is not a directory")
	}

	c := &collection{basePath: repoDir}
	err = c.loadAll()
	return c, err
}

func (c *collection) List(ctx context.Context, pid profile.ID, lp params.List) ([]Item, error) {
	c.Lock()
	defer c.Unlock()

	col, ok := c.collections[pid]
	if !ok {
		return []Item{}, nil
	}

	if lp.Limit < 0 {
		lp.Limit = len(col)
	}

	results := make([]Item, 0, lp.Limit)

	for _, item := range col {
		lp.Offset--
		if lp.Offset > 0 {
			continue
		}

		results = append(results, item)
	}

	return results, nil
}

func (c *collection) Put(ctx context.Context, pid profile.ID, items ...Item) error {
	c.Lock()
	defer c.Unlock()

	for _, item := range items {
		if err := c.putOne(pid, item); err != nil {
			return err
		}
	}
	return c.saveProfileCollection(pid)
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

	c.collections[pid] = append(c.collections[pid], item)
	return nil
}

func (c *collection) Delete(ctx context.Context, pid profile.ID, ids ...string) error {
	c.Lock()
	defer c.Unlock()

	return fmt.Errorf("not finished")
}

func (c *collection) loadAll() error {
	f, err := os.Open(c.basePath)
	if err != nil {
		return err
	}

	names, err := f.Readdirnames(-1)
	if err != nil {
		return err
	}

	c.collections = make(map[profile.ID][]Item)

	for _, filename := range names {
		if isCollectionFilename(filename) {
			if err := c.loadProfileCollection(filename); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *collection) loadProfileCollection(filename string) error {
	pid, err := profile.IDB58Decode(strings.TrimSuffix(filename, ".json"))
	if err != nil {
		return fmt.Errorf("decoding profile ID: %w", err)
	}

	f, err := os.Open(filepath.Join(c.basePath, filename))
	if err != nil {
		return err
	}
	defer f.Close()

	items := []Item{}
	if err := json.NewDecoder(f).Decode(&items); err != nil {
		return err
	}

	c.collections[pid] = items
	return nil
}

func (c *collection) saveProfileCollection(pid profile.ID) error {
	if c.basePath == "" {
		return nil
	}

	items := c.collections[pid]
	if items == nil {
		return fmt.Errorf("cannot save empty collection")
	}

	path := filepath.Join(c.basePath, fmt.Sprintf("%s.json", pid.String()))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0655)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(items)
}

func isCollectionFilename(filename string) bool {
	return strings.HasSuffix(filename, ".json")
}
