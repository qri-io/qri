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

	"github.com/qri-io/qri/dsref"
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
	List(ctx context.Context, pid profile.ID, lp params.List) ([]dsref.VersionInfo, error)
}

// Writable is an extension interface for collection that adds methods for
// adding and removing items
type Writable interface {
	Collection
	Put(ctx context.Context, profileID profile.ID, items ...dsref.VersionInfo) error
	Delete(ctx context.Context, profileID profile.ID, initIDs ...string) error
}

const collectionsDirName = "collections"

type collection struct {
	basePath string

	sync.Mutex  // collections map lock
	collections map[profile.ID][]dsref.VersionInfo
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
			collections: make(map[profile.ID][]dsref.VersionInfo),
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
			collections: make(map[profile.ID][]dsref.VersionInfo),
		}, nil
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("collection is not a directory")
	}

	c := &collection{basePath: repoDir}
	err = c.loadAll()
	return c, err
}

func (c *collection) List(ctx context.Context, pid profile.ID, lp params.List) ([]dsref.VersionInfo, error) {
	c.Lock()
	defer c.Unlock()

	col, ok := c.collections[pid]
	if !ok {
		return []dsref.VersionInfo{}, nil
	}

	if lp.Limit < 0 {
		lp.Limit = len(col)
	}

	results := make([]dsref.VersionInfo, 0, lp.Limit)

	for _, item := range col {
		lp.Offset--
		if lp.Offset > 0 {
			continue
		}

		results = append(results, item)
	}

	return results, nil
}

func (c *collection) Put(ctx context.Context, pid profile.ID, items ...dsref.VersionInfo) error {
	c.Lock()
	defer c.Unlock()

	for _, item := range items {
		if err := c.putOne(pid, item); err != nil {
			return err
		}
	}

	agg, _ := dsref.NewVersionInfoAggregator([]string{"name"})
	agg.Sort(c.collections[pid])

	return c.saveProfileCollection(pid)
}

func (c *collection) putOne(pid profile.ID, item dsref.VersionInfo) error {
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

func (c *collection) Delete(ctx context.Context, pid profile.ID, initID ...string) error {
	c.Lock()
	defer c.Unlock()

	col, ok := c.collections[pid]
	if !ok {
		return fmt.Errorf("no collection for profile")
	}

	for _, removeID := range initID {
		found := false
		for i, item := range col {
			if item.InitID == removeID {
				found = true
				copy(col[i:], col[i+1:])              // Shift a[i+1:] left one index.
				col[len(col)-1] = dsref.VersionInfo{} // Erase last element (write zero value).
				col = col[:len(col)-1]                // Truncate slice.
				break
			}
		}

		if !found {
			fmt.Println("can't find id", removeID)
			return fmt.Errorf("no dataset in collection with initID %q", removeID)
		}
	}

	c.collections[pid] = col

	return c.saveProfileCollection(pid)
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

	c.collections = make(map[profile.ID][]dsref.VersionInfo)

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

	items := []dsref.VersionInfo{}
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
