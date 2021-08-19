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
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
)

const collectionsDirName = "collections"

var (
	// ErrNotFound indicates a query for an unknown value
	ErrNotFound = fmt.Errorf("not found")
	log         = logger.Logger("collection")
)

// SetMaintainer maintains a set of collections, each scoped to a user profile
// It keeps the collection set in sync as subsystems mutate their state.
type SetMaintainer struct {
	set Set
}

// NewSetMaintainer constructs a SetMaintainer
func NewSetMaintainer(ctx context.Context, bus event.Bus, set Set) (*SetMaintainer, error) {
	log.Debugw("NewSetMaintainer")

	if bus == nil {
		return nil, fmt.Errorf("bus of type event.Bus required")
	}
	if set == nil {
		return nil, fmt.Errorf("set of type collection.Set required")
	}
	c := &SetMaintainer{
		set: set,
	}
	c.subscribe(bus)
	return c, nil
}

// Set returns the SetMaintainer's underlying `Set`
func (sm *SetMaintainer) Set() Set {
	return sm.set
}

func (sm *SetMaintainer) subscribe(bus event.Bus) {
	bus.SubscribeTypes(sm.handleEvent,
		// save events
		event.ETDatasetNameInit,
		event.ETDatasetCommitChange,
		event.ETDatasetRename,
		event.ETDatasetDeleteAll,

		// remote & registry events
		event.ETDatasetPushed,
		event.ETDatasetPulled,
		event.ETRegistryProfileCreated,

		// automation events
		event.ETAutomationWorkflowStarted,
		event.ETAutomationWorkflowStopped,
		event.ETAutomationWorkflowCreated,
		event.ETAutomationWorkflowRemoved,

		// fsi
		event.ETFSICreateLink,
		event.ETFSIRemoveLink,
	)
}

func (sm *SetMaintainer) handleEvent(ctx context.Context, e event.Event) error {
	switch e.Type {
	case event.ETDatasetNameInit:
		if vi, ok := e.Payload.(dsref.VersionInfo); ok {
			pid, err := profile.IDB58Decode(vi.ProfileID)
			if err != nil {
				log.Debugw("parsing profile ID in name init", "err", err)
				return err
			}
			if err := sm.set.Add(ctx, pid, vi); err != nil {
				log.Debugw("putting one:", "err", err)
				return err
			}
			log.Debugw("finished putting new name", "name", vi.Name, "initID", vi.InitID)
		}
	case event.ETDatasetCommitChange:
		// keep in mind commit changes can mean added OR removed versions
		if vi, ok := e.Payload.(dsref.VersionInfo); ok {
			sm.set.UpdateEverywhere(ctx, vi.InitID, func(m *dsref.VersionInfo) {
				// preserve fsi path
				vi.FSIPath = m.FSIPath
				// preserve workflow id
				vi.WorkflowID = m.WorkflowID
				// preserve run information
				vi.RunID = m.RunID
				vi.RunStatus = m.RunStatus
				*m = vi
			})
		}
	case event.ETDatasetRename:
		if rename, ok := e.Payload.(event.DsRename); ok {
			sm.set.UpdateEverywhere(ctx, rename.InitID, func(vi *dsref.VersionInfo) {
				vi.Name = rename.NewName
			})
		}
	case event.ETDatasetDeleteAll:
		if e.ProfileID != "" {
			pid, err := profile.IDB58Decode(e.ProfileID)
			if err != nil {
				log.Debugw("parsing profile ID in dataset delete all event", "err", err)
				return err
			}
			if initID, ok := e.Payload.(string); ok {
				if err := sm.set.Delete(ctx, pid, initID); err != nil {
					log.Debugw("removing dataset from collection", "profileID", pid, "initID", initID, "err", err)
				}
			}
		}
	case event.ETRegistryProfileCreated:
		if p, ok := e.Payload.(event.RegistryProfileCreated); ok {
			pid, err := profile.IDB58Decode(p.ProfileID)
			if err != nil {
				log.Debugw("parsing profile ID in registry profile created event", "err", err)
				return err
			}
			sm.set.RenameUsername(ctx, pid, p.Username)
		}
	case event.ETDatasetPushed:
		log.Errorw("need user-scoped events to allow `ETDatasetPush` event to add dataset to a user's collection", e.Payload)
	case event.ETDatasetPulled:
		if e.ProfileID != "" {
			pid, err := profile.IDB58Decode(e.ProfileID)
			if err != nil {
				log.Debugw("parsing profile ID in dataset pulled event", "err", err)
				return err
			}
			if vi, ok := e.Payload.(dsref.VersionInfo); ok {
				if err := sm.set.Add(ctx, pid, vi); err != nil {
					log.Debugw("adding dataset to collection", "profileID", pid, "initID", vi.InitID, "err", err)
				}
			}
		}
	case event.ETAutomationWorkflowStarted:
		if evt, ok := e.Payload.(event.WorkflowStartedEvent); ok {
			err := sm.set.UpdateEverywhere(ctx, evt.InitID, func(vi *dsref.VersionInfo) {
				vi.RunID = evt.RunID
				vi.RunStatus = "running"
			})
			if err != nil {
				log.Debugw("updating dataset across all collections", "InitID", evt.InitID, "err", err)
			}
		}
	case event.ETAutomationWorkflowStopped:
		if evt, ok := e.Payload.(event.WorkflowStoppedEvent); ok {
			err := sm.set.UpdateEverywhere(ctx, evt.InitID, func(vi *dsref.VersionInfo) {
				vi.RunStatus = evt.Status
			})
			if err != nil {
				log.Debugw("updating dataset across all collections", "InitID", evt.InitID, "err", err)
			}
		}
	case event.ETAutomationWorkflowCreated:
		if wf, ok := e.Payload.(workflow.Workflow); ok {
			err := sm.set.UpdateEverywhere(ctx, wf.InitID, func(vi *dsref.VersionInfo) {
				vi.WorkflowID = wf.WorkflowID()
			})

			if err != nil {
				log.Debugw("updating dataset across all collections", "InitID", wf.InitID, "err", err)
			}
		}
	case event.ETAutomationWorkflowRemoved:
		if wf, ok := e.Payload.(workflow.Workflow); ok {
			err := sm.set.UpdateEverywhere(ctx, wf.InitID, func(vi *dsref.VersionInfo) {
				vi.WorkflowID = ""
			})

			if err != nil {
				log.Debugw("updating dataset across all collections", "InitID", wf.InitID, "err", err)
			}
		}
	case event.ETFSICreateLink:
		if link, ok := e.Payload.(event.FSICreateLink); ok {
			sm.set.UpdateEverywhere(ctx, link.InitID, func(vi *dsref.VersionInfo) {
				vi.FSIPath = link.FSIPath
			})
		}
	case event.ETFSIRemoveLink:
		if change, ok := e.Payload.(event.FSIRemoveLink); ok {
			sm.set.UpdateEverywhere(ctx, change.InitID, func(vi *dsref.VersionInfo) {
				vi.FSIPath = ""
			})
		}
	}
	return nil
}

// Set maintains lists of dataset information, called a collection, with each
// list scoped to a user profile. A user's collection may consist of information
// from datasets they have created and datasets added from other users.
// Collections are the canonical source of truth for listing a user's datasets
// in a qri instance. While a collection owns the list, the fields in a
// collection item are cached values gathered from other subsystems, and must
// be kept in sync as subsystems mutate their state
type Set interface {
	// List the collection of a single user
	List(ctx context.Context, pid profile.ID, lp params.List) ([]dsref.VersionInfo, error)
	// Get info about a single dataset in a single user's collection
	Get(ctx context.Context, pid profile.ID, initID string) (dsref.VersionInfo, error)
	// PutList adds a list of `dsref.VersionInfo`s to a user's collection
	PutList(ctx context.Context, pid profile.ID, items []dsref.VersionInfo) error
	// Add adds a single dataset to a user's collection
	Add(ctx context.Context, pid profile.ID, add dsref.VersionInfo) error
	// RenameUsername changes a user's name
	RenameUsername(ctx context.Context, pid profile.ID, newUsername string) error
	// UpdateEverywhere updates a dataset in all collections that contain it
	UpdateEverywhere(ctx context.Context, initID string, mutate func(vi *dsref.VersionInfo)) error
	// Delete removes a single dataset from a single user's collection
	Delete(ctx context.Context, pid profile.ID, removeID string) error
}

// LocalSetOptionFunc passes an options pointer for confugration during
// LocalSet construction
type LocalSetOptionFunc func(o *LocalSetOptions)

// LocalSetOptions configures local set runtime behaviour
type LocalSetOptions struct {
	// If MigrateRepo is provided & a local colleciton doesn't exist, LocalSet
	// will run a migration, creating a new set from the provided repo, creating
	// a collection set for the "root" user
	MigrateRepo repo.Repo
}

type localSet struct {
	basePath string

	sync.Mutex  // collections map lock
	collections map[profile.ID][]dsref.VersionInfo
}

var (
	_ Set = (*localSet)(nil)
)

// NewLocalSet constructs a node-local collection set. If repoDir is not the
// empty string, localCollection will create a "collections" directory to
// persist collections, serializing to a directory of "profileID.json" files,
// with one for each profileID in the set of collections. providing an empty
// repoDir value will create an in-memory collection
func NewLocalSet(ctx context.Context, repoDir string, options ...LocalSetOptionFunc) (Set, error) {
	opt := &LocalSetOptions{}
	for _, fn := range options {
		fn(opt)
	}

	if repoDir == "" {
		// in-memory only collection
		s := &localSet{
			collections: make(map[profile.ID][]dsref.VersionInfo),
		}
		if opt.MigrateRepo != nil {
			if err := MigrateRepoStoreToLocalCollectionSet(ctx, s, opt.MigrateRepo); err != nil {
				return nil, err
			}
		}
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

		if opt.MigrateRepo != nil {
			if err = MigrateRepoStoreToLocalCollectionSet(ctx, s, opt.MigrateRepo); err != nil {
				return nil, err
			}
		}
		return s, nil
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("collection is not a directory")
	}

	s := &localSet{basePath: repoDir}

	err = s.loadAll()
	return s, err
}

func (s *localSet) List(ctx context.Context, pid profile.ID, lp params.List) ([]dsref.VersionInfo, error) {
	s.Lock()
	defer s.Unlock()

	if err := pid.Validate(); err != nil {
		return nil, err
	}

	col, ok := s.collections[pid]
	if !ok {
		return nil, fmt.Errorf("%w: no collection for profile ID %q", ErrNotFound, pid.Encode())
	}

	if lp.Limit < 0 {
		lp.Limit = len(col)
	}

	results := make([]dsref.VersionInfo, 0, lp.Limit)

	for _, item := range col {
		lp.Offset--
		if lp.Offset >= 0 {
			continue
		}

		results = append(results, item)
		if len(results) == lp.Limit {
			break
		}
	}

	return results, nil
}

func (s *localSet) Get(ctx context.Context, pid profile.ID, initID string) (dsref.VersionInfo, error) {
	s.Lock()
	defer s.Unlock()

	collection, ok := s.collections[pid]
	if !ok {
		return dsref.VersionInfo{}, fmt.Errorf("%w: no collection for profile ID %q", ErrNotFound, pid.Encode())
	}
	for _, vi := range collection {
		if vi.InitID == initID {
			return vi, nil
		}
	}
	return dsref.VersionInfo{}, ErrNotFound
}

func (s *localSet) PutList(ctx context.Context, pid profile.ID, items []dsref.VersionInfo) error {
	s.Lock()
	defer s.Unlock()

	if err := pid.Validate(); err != nil {
		return err
	}

	for _, item := range items {
		if err := s.add(pid, item); err != nil {
			return err
		}
	}

	agg, _ := dsref.NewVersionInfoAggregator([]string{"name"})
	agg.Sort(s.collections[pid])

	return s.saveProfileCollection(pid)
}

func (s *localSet) Add(ctx context.Context, pid profile.ID, item dsref.VersionInfo) error {
	s.Lock()
	defer s.Unlock()

	if err := pid.Validate(); err != nil {
		return err
	}

	if err := s.add(pid, item); err != nil {
		return err
	}
	agg, _ := dsref.NewVersionInfoAggregator([]string{"name"})
	agg.Sort(s.collections[pid])

	return s.saveProfileCollection(pid)
}

func (s *localSet) add(pid profile.ID, item dsref.VersionInfo) error {
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

func (s *localSet) Delete(ctx context.Context, pid profile.ID, removeID string) error {
	s.Lock()
	defer s.Unlock()

	if err := pid.Validate(); err != nil {
		return err
	}

	col, ok := s.collections[pid]
	if !ok {
		return fmt.Errorf("no collection for profile")
	}

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
		items = []dsref.VersionInfo{}
	}

	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("serializing user collection: %w", err)
	}

	path := filepath.Join(s.basePath, fmt.Sprintf("%s.json", pid.Encode()))
	return ioutil.WriteFile(path, data, 0644)
}

func (s *localSet) RenameUsername(ctx context.Context, pid profile.ID, newUsername string) error {
	s.Lock()
	defer s.Unlock()

	for profileID, col := range s.collections {
		changed := false
		for i, vi := range col {
			if vi.ProfileID == pid.Encode() && vi.Username != newUsername {
				changed = true
				vi.Username = newUsername
				col[i] = vi
			}
		}
		if changed {
			if err := s.saveProfileCollection(profileID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *localSet) UpdateEverywhere(ctx context.Context, initID string, mutate func(vi *dsref.VersionInfo)) error {
	s.Lock()
	defer s.Unlock()

	for pid, col := range s.collections {
		for i, vi := range col {
			if vi.InitID == initID {
				mutate(&vi)
				col[i] = vi
				s.collections[pid] = col
				if err := s.saveProfileCollection(pid); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func isCollectionFilename(filename string) bool {
	return strings.HasSuffix(filename, ".json")
}
