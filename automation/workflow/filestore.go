package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/profile"
)

// fileStore is a store implementation that writes to a file of JSON bytes.
// fileStore is safe for concurrent use
type fileStore struct {
	path      string
	lock      sync.Mutex
	workflows *Set
}

// compile-time assertion that fileStore is a Store
var _ Store = (*fileStore)(nil)

// NewFileStore creates a workflow store that persists to a file
func NewFileStore(repoPath string) (Store, error) {
	s := &fileStore{
		path:      filepath.Join(repoPath, "workflows.json"),
		workflows: NewSet(),
	}

	return s, s.loadFromFile()
}

// ListWorkflows lists workflows currently in the store
func (s *fileStore) List(ctx context.Context, pid profile.ID, lp params.List) ([]*Workflow, error) {
	fetchAll := false
	switch {
	case lp.Limit == -1 && lp.Offset == 0:
		fetchAll = true
	case lp.Limit < 0:
		return nil, fmt.Errorf("limit of %d is out of bounds", lp.Limit)
	case lp.Offset < 0:
		return nil, fmt.Errorf("offset of %d is out of bounds", lp.Offset)
	case lp.Limit == 0 || lp.Offset > s.workflows.Len():
		return []*Workflow{}, nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	start := lp.Offset
	end := lp.Offset + lp.Limit
	if end > s.workflows.Len() || fetchAll {
		end = s.workflows.Len()
	}

	sort.Sort(s.workflows)
	return s.workflows.Slice(start, end), nil
}

// ListWorkflowsByStatus lists workflows filtered by status and ordered in reverse
// chronological order by `LatestStart`
func (s *fileStore) ListDeployed(ctx context.Context, pid profile.ID, lp params.List) ([]*Workflow, error) {
	deployed := NewSet()
	fetchAll := false
	switch {
	case lp.Limit == -1 && lp.Offset == 0:
		fetchAll = true
	case lp.Limit < 0:
		return nil, fmt.Errorf("limit of %d is out of bounds", lp.Limit)
	case lp.Offset < 0:
		return nil, fmt.Errorf("offset of %d is out of bounds", lp.Offset)
	case lp.Limit == 0:
		return []*Workflow{}, nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, wf := range s.workflows.set {
		if wf.Active {
			deployed.Add(wf)
		}
	}

	if lp.Offset >= deployed.Len() {
		return []*Workflow{}, nil
	}

	start := lp.Offset
	end := lp.Offset + lp.Limit
	if end > deployed.Len() || fetchAll {
		end = deployed.Len()
	}

	sort.Sort(deployed)
	return deployed.Slice(start, end), nil
}

// GetWorkflowByInitID gets a workflow with the corresponding InitID field
func (s *fileStore) GetByInitID(ctx context.Context, initID string) (*Workflow, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, workflow := range s.workflows.set {
		if workflow.InitID == initID {
			return workflow, nil
		}
	}
	return nil, ErrNotFound
}

// GetWorkflow gets workflow details from the store by dataset identifier
func (s *fileStore) Get(ctx context.Context, id ID) (*Workflow, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, workflow := range s.workflows.set {
		if workflow.ID == id {
			return workflow, nil
		}
	}
	return nil, ErrNotFound
}

// PutWorkflow places a workflow in the store. If the workflow name matches the name of a workflow
// that already exists, it will be overwritten with the new workflow
func (s *fileStore) Put(ctx context.Context, wf *Workflow) (*Workflow, error) {
	if wf == nil {
		return nil, ErrNilWorkflow
	}
	w := wf.Copy()
	if wf.ID == "" {
		if _, err := s.GetByInitID(ctx, w.InitID); !errors.Is(err, ErrNotFound) {
			return nil, ErrWorkflowForDatasetExists
		}
		w.ID = NewID()
	}
	if err := w.Validate(); err != nil {
		return nil, err
	}
	s.lock.Lock()
	s.workflows.Add(w)
	s.lock.Unlock()

	return w, s.writeToFile()
}

// DeleteWorkflow removes a workflow from the store by name. deleting a non-existent workflow
// won't return an error
func (s *fileStore) Remove(ctx context.Context, id ID) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if removed := s.workflows.Remove(id); removed {
		return s.writeToFileNoLock()
	}
	return ErrNotFound
}

// Shutdown writes the set of workflows to the filestore
func (s *fileStore) Shutdown(ctx context.Context) error {
	return s.writeToFile()
}

func (s *fileStore) loadFromFile() (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		log.Debugw("fileStore loading store from file", "error", err)
		return err
	}

	state := struct {
		Workflows *Set
	}{}
	if err := json.Unmarshal(data, &state); err != nil {
		log.Debugw("fileStore deserializing from JSON", "error", err)
		return err
	}

	if state.Workflows != nil {
		s.workflows = state.Workflows
	}
	return nil
}

func (s *fileStore) writeToFile() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.writeToFileNoLock()
}

// Only use this when you have a surrounding lock
func (s *fileStore) writeToFileNoLock() error {
	state := struct {
		Workflows *Set `json:"workflows"`
	}{
		Workflows: s.workflows,
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.path, data, 0644)
}
