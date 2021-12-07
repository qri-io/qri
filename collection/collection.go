// Package collection maintains a list of user datasets
package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

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
	Set
}

// NewSetMaintainer constructs a SetMaintainer
func NewSetMaintainer(ctx context.Context, bus event.Bus, set Set) (*SetMaintainer, error) {
	if bus == nil {
		return nil, fmt.Errorf("bus of type event.Bus required")
	}
	if set == nil {
		return nil, fmt.Errorf("set of type collection.Set required")
	}
	c := &SetMaintainer{
		Set: set,
	}
	c.subscribe(bus)
	return c, nil
}

func (sm *SetMaintainer) subscribe(bus event.Bus) {
	bus.SubscribeTypes(sm.handleEvent,
		// save events
		event.ETDatasetNameInit,
		event.ETDatasetRename,
		event.ETDatasetDeleteAll,
		event.ETDatasetDownload,

		// remote & registry events
		event.ETDatasetPushed,
		event.ETDatasetPulled,
		event.ETRegistryProfileCreated,
		event.ETRemoteDatasetFollowed,
		event.ETRemoteDatasetUnfollowed,
		event.ETRemoteDatasetIssueOpened,
		event.ETRemoteDatasetIssueClosed,

		// automation events
		event.ETAutomationWorkflowCreated,
		event.ETAutomationWorkflowRemoved,

		// transform events
		event.ETTransformStart,
		event.ETTransformCanceled,

		// log events
		event.ETLogbookWriteCommit,
		event.ETLogbookWriteRun,
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
			if err := sm.Add(ctx, pid, vi); err != nil {
				log.Debugw("putting one:", "err", err)
				return err
			}
			log.Debugw("finished putting new name", "name", vi.Name, "initID", vi.InitID)
		}
	case event.ETLogbookWriteCommit:
		// keep in mind commit changes can mean added OR removed versions
		if vi, ok := e.Payload.(dsref.VersionInfo); ok {
			sm.UpdateEverywhere(ctx, vi.InitID, func(m *dsref.VersionInfo) {
				// preserve fields that are not tracked in `ETLogbookWriteCommit`
				vi.WorkflowID = m.WorkflowID
				vi.DownloadCount = m.DownloadCount
				vi.RunCount = m.RunCount
				vi.FollowerCount = m.FollowerCount
				vi.OpenIssueCount = m.OpenIssueCount

				// preserve "last run" information
				if vi.RunID == "" {
					vi.RunID = m.RunID
					vi.RunStatus = m.RunStatus
					vi.RunDuration = m.RunDuration
					vi.RunStart = m.RunStart
				}

				*m = vi
			})
		}
	case event.ETDatasetRename:
		if rename, ok := e.Payload.(event.DsRename); ok {
			sm.UpdateEverywhere(ctx, rename.InitID, func(vi *dsref.VersionInfo) {
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
				if err := sm.Delete(ctx, pid, initID); err != nil {
					log.Debugw("removing dataset from collection", "profileID", pid, "initID", initID, "err", err)
				}
			}
		}
	case event.ETDatasetDownload:
		if initID, ok := e.Payload.(string); ok {
			sm.UpdateEverywhere(ctx, initID, func(vi *dsref.VersionInfo) {
				vi.DownloadCount++
			})
		}
	case event.ETRegistryProfileCreated:
		if p, ok := e.Payload.(event.RegistryProfileCreated); ok {
			pid, err := profile.IDB58Decode(p.ProfileID)
			if err != nil {
				log.Debugw("parsing profile ID in registry profile created event", "err", err)
				return err
			}
			sm.RenameUser(ctx, pid, p.Username)
		}
	case event.ETDatasetPushed:
		log.Errorw("no handler for event `ETDatasetPush`")
	case event.ETDatasetPulled:
		if e.ProfileID != "" {
			pid, err := profile.IDB58Decode(e.ProfileID)
			if err != nil {
				log.Debugw("parsing profile ID in dataset pulled event", "err", err)
				return err
			}
			if vi, ok := e.Payload.(dsref.VersionInfo); ok {
				if err := sm.Add(ctx, pid, vi); err != nil {
					log.Debugw("adding dataset to collection", "profileID", pid, "initID", vi.InitID, "err", err)
				}
			}
		}
	case event.ETRemoteDatasetFollowed:
		if initID, ok := e.Payload.(string); ok {
			sm.UpdateEverywhere(ctx, initID, func(vi *dsref.VersionInfo) {
				vi.FollowerCount++
			})
		}
	case event.ETRemoteDatasetUnfollowed:
		if initID, ok := e.Payload.(string); ok {
			sm.UpdateEverywhere(ctx, initID, func(vi *dsref.VersionInfo) {
				vi.FollowerCount--
				if vi.FollowerCount < 0 {
					vi.FollowerCount = 0
				}
			})
		}
	case event.ETRemoteDatasetIssueOpened:
		if initID, ok := e.Payload.(string); ok {
			sm.UpdateEverywhere(ctx, initID, func(vi *dsref.VersionInfo) {
				vi.OpenIssueCount++
			})
		}
	case event.ETRemoteDatasetIssueClosed:
		if initID, ok := e.Payload.(string); ok {
			sm.UpdateEverywhere(ctx, initID, func(vi *dsref.VersionInfo) {
				vi.OpenIssueCount--
				if vi.OpenIssueCount < 0 {
					vi.OpenIssueCount = 0
				}
			})
		}
	case event.ETAutomationWorkflowCreated:
		if wf, ok := e.Payload.(workflow.Workflow); ok {
			err := sm.UpdateEverywhere(ctx, wf.InitID, func(vi *dsref.VersionInfo) {
				vi.WorkflowID = wf.WorkflowID()
			})

			if err != nil {
				log.Debugw("updating dataset across all collections", "InitID", wf.InitID, "err", err)
			}
		}
	case event.ETAutomationWorkflowRemoved:
		if wf, ok := e.Payload.(workflow.Workflow); ok {
			err := sm.UpdateEverywhere(ctx, wf.InitID, func(vi *dsref.VersionInfo) {
				vi.WorkflowID = ""
			})

			if err != nil {
				log.Debugw("updating dataset across all collections", "InitID", wf.InitID, "err", err)
			}
		}
	case event.ETTransformStart:
		if te, ok := e.Payload.(event.TransformLifecycle); ok {
			if te.Mode != "apply" {
				err := sm.UpdateEverywhere(ctx, te.InitID, func(vi *dsref.VersionInfo) {
					vi.RunCount++
					vi.RunStatus = "running"
					vi.RunID = te.RunID
					vi.RunStart = nil
				})
				if err != nil {
					log.Debugw("update dataset across all collections", "InitID", te.InitID, "err", err)
				}
			}
		}
	case event.ETLogbookWriteRun:
		if vi, ok := e.Payload.(dsref.VersionInfo); ok {
			err := sm.UpdateEverywhere(ctx, vi.InitID, func(v *dsref.VersionInfo) {
				v.RunStart = vi.RunStart
				v.RunID = vi.RunID
				v.RunStatus = vi.RunStatus
				v.RunDuration = vi.RunDuration
			})
			if err != nil {
				log.Debugw("update dataset across all collections", "InitID", vi.InitID, "err", err)
			}
		}
	case event.ETTransformCanceled:
		if te, ok := e.Payload.(event.TransformLifecycle); ok {
			err := sm.UpdateEverywhere(ctx, te.InitID, func(v *dsref.VersionInfo) {
				v.RunStatus = "failed"
			})
			if err != nil {
				log.Debugw("update dataset across all collections", "InitID", te.InitID, "err", err)
			}
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
	Get(ctx context.Context, pid profile.ID, initID string) (*dsref.VersionInfo, error)
	// Add adds a dataset or datasets to a user's collection
	Add(ctx context.Context, pid profile.ID, add ...dsref.VersionInfo) error
	// RenameUser changes a user's name
	RenameUser(ctx context.Context, pid profile.ID, newUsername string) error
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

// compile-time assertion for type interface
var _ Set = (*localSet)(nil)

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

// List currently supports ordering by name (ascending) and last update (descending)
// default is name
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

	if len(lp.OrderBy) != 0 {
		switch lp.OrderBy[0].Key {
		case "updated":
			sortedCol := make([]dsref.VersionInfo, len(col))
			copy(sortedCol, col)
			sort.Sort(byUpdated(sortedCol))
			col = sortedCol
		case "name":
		default:
		}
	}

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

func (s *localSet) Get(ctx context.Context, pid profile.ID, initID string) (*dsref.VersionInfo, error) {
	s.Lock()
	defer s.Unlock()

	if err := pid.Validate(); err != nil {
		return nil, err
	}

	collection, ok := s.collections[pid]
	if !ok {
		return nil, fmt.Errorf("%w: no collection for profile ID %q", ErrNotFound, pid.Encode())
	}
	for _, vi := range collection {
		if vi.InitID == initID {
			return &vi, nil
		}
	}
	return nil, ErrNotFound
}

func (s *localSet) Add(ctx context.Context, pid profile.ID, items ...dsref.VersionInfo) error {
	s.Lock()
	defer s.Unlock()

	if err := pid.Validate(); err != nil {
		return err
	}

	for _, item := range items {
		if err := s.addOne(pid, item); err != nil {
			return err
		}
	}
	agg, _ := dsref.NewVersionInfoAggregator([]string{"name"})
	agg.Sort(s.collections[pid])

	return s.saveProfileCollection(pid)
}

func (s *localSet) addOne(pid profile.ID, item dsref.VersionInfo) error {
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

func (s *localSet) RenameUser(ctx context.Context, pid profile.ID, newUsername string) error {
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

// byUpdate sorts by updated time (commit or run), decending
type byUpdated []dsref.VersionInfo

func (u byUpdated) Len() int      { return len(u) }
func (u byUpdated) Swap(i, j int) { u[i], u[j] = u[j], u[i] }
func (u byUpdated) Less(i, j int) bool {
	ivi := u[i]
	jvi := u[j]
	iTime := ivi.CommitTime
	jTime := jvi.CommitTime

	if ivi.RunStart != nil {
		iTime = ivi.RunStart.Add(time.Duration(ivi.RunDuration))
	}
	if jvi.RunStart != nil {
		jTime = jvi.RunStart.Add(time.Duration(jvi.RunDuration))
	}

	return iTime.After(jTime)
}
