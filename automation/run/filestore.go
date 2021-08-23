package run

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
)

type fileStore struct {
	path  string
	store *MemStore
}

// compile-time assertion that fileStore is a Store
var _ Store = (*fileStore)(nil)

// NewFileStore creates a workflow store that persists to a file
func NewFileStore(repoPath string) (Store, error) {
	s := &fileStore{
		path:  filepath.Join(repoPath, "runs.json"),
		store: NewMemStore(),
	}

	return s, s.loadFromFile()
}

func (s *fileStore) loadFromFile() error {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		log.Debugw("fileStore loading store from file", "error", err)
		return err
	}
	if err := json.Unmarshal(data, s.store); err != nil {
		log.Debugw("fileStore deserializing from JSON", "error", err)
		return err
	}

	return nil
}

func (s *fileStore) writeToFile() error {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	return s.writeToFileNoLock()
}

// Only use this when you have a surrounding lock
func (s *fileStore) writeToFileNoLock() error {
	data, err := json.Marshal(s.store)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.path, data, 0644)
}

// Create adds a new run State to the Store
func (s *fileStore) Create(ctx context.Context, r *State) (*State, error) {
	return s.store.Create(ctx, r)
}

// Put puts a run State with an existing run ID into the Store
func (s *fileStore) Put(ctx context.Context, r *State) (*State, error) { return s.store.Put(ctx, r) }

// Get gets the associated run.State
func (s *fileStore) Get(ctx context.Context, id string) (*State, error) { return s.store.Get(ctx, id) }

// Count returns the number of runs for a given workflow.ID
func (s *fileStore) Count(ctx context.Context, wid workflow.ID) (int, error) {
	return s.store.Count(ctx, wid)
}

// List lists all the runs associated with the workflow.ID in reverse
// chronological order
func (s *fileStore) List(ctx context.Context, wid workflow.ID, lp params.List) ([]*State, error) {
	return s.store.List(ctx, wid, lp)
}

// GetLatest returns the most recent run associated with the workflow id
func (s *fileStore) GetLatest(ctx context.Context, wid workflow.ID) (*State, error) {
	return s.store.GetLatest(ctx, wid)
}

// GetStatus returns the status of the latest run based on the
// workflow.ID
func (s *fileStore) GetStatus(ctx context.Context, wid workflow.ID) (Status, error) {
	return s.store.GetStatus(ctx, wid)
}

// ListByStatus returns a list of run.State entries with a given status
// looking only at the most recent run of each Workflow
func (s *fileStore) ListByStatus(ctx context.Context, owner profile.ID, status Status, lp params.List) ([]*State, error) {
	return s.store.ListByStatus(ctx, owner, status, lp)
}

// Shutdown writes the run events to the filestore
func (s *fileStore) Shutdown() error {
	if err := s.writeToFile(); err != nil {
		return err
	}
	return s.store.Shutdown()
}

// AddEvent writes an event to the store, attaching it to an existing stored
// run state
func (s *fileStore) AddEvent(id string, e event.Event) error {
	return s.store.AddEvent(id, e)
}
