// Package collection maintains a list of user datasets
package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	logger "github.com/ipfs/go-log"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
)

var (
	// ErrNotFound indicates a query for an unknown value
	ErrNotFound = fmt.Errorf("not found")
	log         = logger.Logger("collection")
)

// Set maintains lists of datasets, with each list scoped to a user profile.
// A user's collection may consist of datasets they have created and datasets
// added from other users. Collections are the canonical source of truth for
// listing a user's datasets in a qri instance. While a collection owns the list
// the fields in a collection item are cached values gathered from other
// subsystems, and must be kept in sync as subsystems mutate their state.
type Set interface {
	List(ctx context.Context, pid profile.ID, lp params.List) ([]dsref.VersionInfo, error)
}

// WritableSet is an extension interface on Set that adds methods for
// adding and removing items
type WritableSet interface {
	Set
	Put(ctx context.Context, profileID profile.ID, items ...dsref.VersionInfo) error
	Delete(ctx context.Context, profileID profile.ID, initIDs ...string) error
}

const collectionsDirName = "collections"

type localSet struct {
	basePath string

	sync.Mutex  // collections map lock
	collections map[profile.ID][]dsref.VersionInfo
}

var (
	_ Set         = (*localSet)(nil)
	_ WritableSet = (*localSet)(nil)
)

// NewLocalSet constructs a node-local collection set. If repoDir is not the
// empty string, localCollection will create a "collections" directory to
// persist collections, serializing to a directory of "profileID.json" files,
// with one for each profileID in the set of collections. providing an empty
// repoDir value will create an in-memory collection
func NewLocalSet(ctx context.Context, bus event.Bus, repoDir string) (Set, error) {
	if repoDir == "" {
		// in-memory only collection
		s := &localSet{
			collections: make(map[profile.ID][]dsref.VersionInfo),
		}
		s.subscribe(bus)
		return s, nil
	}

	repoDir = filepath.Join(repoDir, collectionsDirName)
	fi, err := os.Stat(repoDir)
	if os.IsNotExist(err) {
		if err := os.Mkdir(repoDir, 0755); err != nil {
			return nil, fmt.Errorf("creating collection directory: %w", err)
		}
		s := &localSet{
			basePath:    repoDir,
			collections: make(map[profile.ID][]dsref.VersionInfo),
		}
		s.subscribe(bus)
		return s, nil
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("collection is not a directory")
	}

	s := &localSet{basePath: repoDir}
	s.subscribe(bus)

	err = s.loadAll()
	return s, err
}

func (s *localSet) List(ctx context.Context, pid profile.ID, lp params.List) ([]dsref.VersionInfo, error) {
	s.Lock()
	defer s.Unlock()

	col, ok := s.collections[pid]
	if !ok {
		return nil, fmt.Errorf("%w: no collection for profile ID %q", ErrNotFound, pid.String())
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

func (s *localSet) Put(ctx context.Context, pid profile.ID, items ...dsref.VersionInfo) error {
	s.Lock()
	defer s.Unlock()

	for _, item := range items {
		if err := s.putOne(pid, item); err != nil {
			return err
		}
	}

	agg, _ := dsref.NewVersionInfoAggregator([]string{"name"})
	agg.Sort(s.collections[pid])

	return s.saveProfileCollection(pid)
}

func (s *localSet) putOne(pid profile.ID, item dsref.VersionInfo) error {
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

	for i, ds := range s.collections[pid] {
		if ds.InitID == item.InitID {
			s.collections[pid][i] = item
			return nil
		}
	}

	s.collections[pid] = append(s.collections[pid], item)
	return nil
}

func (s *localSet) Delete(ctx context.Context, pid profile.ID, initID ...string) error {
	s.Lock()
	defer s.Unlock()

	col, ok := s.collections[pid]
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
			return fmt.Errorf("no dataset in collection with initID %q", removeID)
		}
	}

	s.collections[pid] = col
	return s.saveProfileCollection(pid)
}

func (s *localSet) loadAll() error {
	f, err := os.Open(s.basePath)
	if err != nil {
		return err
	}

	names, err := f.Readdirnames(-1)
	if err != nil {
		return err
	}

	s.collections = make(map[profile.ID][]dsref.VersionInfo)

	for _, filename := range names {
		if isCollectionFilename(filename) {
			if err := s.loadProfileCollection(filename); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *localSet) loadProfileCollection(filename string) error {
	pid, err := profile.IDB58Decode(strings.TrimSuffix(filename, ".json"))
	if err != nil {
		return fmt.Errorf("decoding profile ID: %w", err)
	}

	f, err := os.Open(filepath.Join(s.basePath, filename))
	if err != nil {
		return err
	}
	defer f.Close()

	items := []dsref.VersionInfo{}
	if err := json.NewDecoder(f).Decode(&items); err != nil {
		return err
	}

	s.collections[pid] = items
	return nil
}

func (s *localSet) saveProfileCollection(pid profile.ID) error {
	if s.basePath == "" {
		return nil
	}

	items := s.collections[pid]
	if items == nil {
		return fmt.Errorf("saving collection: %w: profile ID %q", ErrNotFound, pid)
	}

	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("serializing user collection: %w", err)
	}

	path := filepath.Join(s.basePath, fmt.Sprintf("%s.json", pid.String()))
	return ioutil.WriteFile(path, data, 0644)
}

func (s *localSet) subscribe(bus event.Bus) {
	bus.SubscribeTypes(s.handleEvent,
		event.ETDatasetNameInit,
		event.ETDatasetCommitChange,
		event.ETDatasetRename,
		event.ETDatasetDeleteAll,
	)
}

func (s *localSet) handleEvent(ctx context.Context, e event.Event) error {
	switch e.Type {
	case event.ETDatasetNameInit:
		if change, ok := e.Payload.(event.DsChange); ok {
			if vi := change.Info; vi != nil {
				pid, err := profile.NewB58ID(vi.ProfileID)
				if err != nil {
					fmt.Printf("error parsing profile ID in name init: %s\n", err)
					return err
				}
				if err := s.putOne(pid, *change.Info); err != nil {
					fmt.Printf("error putting one: %s\n", err)
					return err
				}
			}
		}
	case event.ETDatasetCommitChange:
		if change, ok := e.Payload.(event.DsChange); ok {
			s.updateOneAcrossAllCollections(change.InitID, func(vi *dsref.VersionInfo) {
				vi.Path = change.HeadRef
			})
		}
	case event.ETDatasetRename:
		if change, ok := e.Payload.(event.DsChange); ok {
			s.updateOneAcrossAllCollections(change.InitID, func(vi *dsref.VersionInfo) {
				vi.Name = change.PrettyName
			})
		}
	case event.ETDatasetDeleteAll:
		// TODO(b5): need user-scoped events for this, unless we want one user
		// removing a dataset to remove it from all collections of all users
		if change, ok := e.Payload.(event.DsChange); ok {
			if err := s.deleteAcrossAllCollections(change.InitID); err != nil {
				// TODO(b5): log error
				log.Debugw("removing dataset across all collections", "initID", change.InitID, "err", err)
			}
		}
	}
	return nil
}

func (s *localSet) updateOneAcrossAllCollections(initID string, mutate func(vi *dsref.VersionInfo)) error {
	s.Lock()
	defer s.Unlock()

	for pid, col := range s.collections {
		for i, vi := range col {
			if vi.InitID == initID {
				mutate(&vi)
				col[i] = vi
				if err := s.saveProfileCollection(pid); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *localSet) deleteAcrossAllCollections(removeID string) error {
	s.Lock()
	defer s.Unlock()

	for pid, col := range s.collections {
		for i, item := range col {
			if item.InitID == removeID {
				copy(col[i:], col[i+1:])              // Shift a[i+1:] left one index.
				col[len(col)-1] = dsref.VersionInfo{} // Erase last element (write zero value).
				s.collections[pid] = col[:len(col)-1] // Truncate slice.
				if err := s.saveProfileCollection(pid); err != nil {
					return err
				}
				break
			}
		}
	}

	return nil
}

func isCollectionFilename(filename string) bool {
	return strings.HasSuffix(filename, ".json")
}
