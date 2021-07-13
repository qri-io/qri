package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
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
func (s *fileStore) List(ctx context.Context, limit, offset int) ([]*Workflow, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if limit < 0 {
		limit = len(s.workflows.set)
	}

	workflows := make([]*Workflow, 0, limit)
	for i, workflow := range s.workflows.set {
		if i < offset {
			continue
		} else if len(workflows) == limit {
			break
		}

		workflows = append(workflows, workflow)
	}
	return workflows, nil
}

// ListWorkflowsByStatus lists workflows filtered by status and ordered in reverse
// chronological order by `LatestStart`
func (s *fileStore) ListDeployed(ctx context.Context, limit, offset int) ([]*Workflow, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	workflows := make([]*Workflow, 0, len(s.workflows.set))

	for _, workflow := range s.workflows.set {
		if workflow.Deployed {
			workflows = append(workflows, workflow)
		}
	}

	if offset > len(workflows) {
		return []*Workflow{}, nil
	}

	if limit < 0 {
		limit = len(workflows)
	}

	if offset+limit > len(workflows) {
		return workflows[offset:], nil
	}

	return workflows[offset:limit], nil
}

// GetWorkflowByName gets a workflow with the corresponding name field. usually matches
// the dataset name
// func (s *fileStore) GetWorkflowByName(ctx context.Context, name string) (*Workflow, error) {
// 	s.lock.Lock()
// 	defer s.lock.Unlock()

// 	for _, workflow := range s.workflows.set {
// 		if workflow.Name == name {
// 			return workflow, nil
// 		}
// 	}
// 	return nil, ErrNotFound
// }

// GetWorkflowByDatasetID gets a workflow with the corresponding datasetID field
func (s *fileStore) GetByDatasetID(datasetID string) (*Workflow, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, workflow := range s.workflows.set {
		if workflow.DatasetID == datasetID {
			return workflow, nil
		}
	}
	return nil, ErrNotFound
}

// GetWorkflow gets workflow details from the store by dataset identifier
func (s *fileStore) Get(id ID) (*Workflow, error) {
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
func (s *fileStore) Put(wf *Workflow) (*Workflow, error) {
	if wf.ID != "" {
		fetchedWF, err := s.Get(wf.ID)
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		if fetchedWF.DatasetID != wf.DatasetID {
			return nil, ErrPutDatasetIDMismatch
		}
		if fetchedWF.OwnerID != wf.OwnerID {
			return nil, ErrPutOwnerIDMismatch
		}
	}
	if wf.ID == "" {
		if _, err := s.GetByDatasetID(wf.DatasetID); !errors.Is(err, ErrNotFound) {
			return nil, ErrWorkflowForDatasetExists
		}
		wf.ID = NewID()
	}
	if err := wf.Validate(); err != nil {
		return nil, err
	}

	s.lock.Lock()
	s.workflows.Add(wf)
	s.lock.Unlock()

	return wf, s.writeToFile()
}

// DeleteWorkflow removes a workflow from the store by name. deleting a non-existent workflow
// won't return an error
func (s *fileStore) Remove(id ID) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if removed := s.workflows.Remove(id); removed {
		return s.writeToFileNoLock()
	}
	return ErrNotFound
}

// DeleteAllWorkflows removes all the workflow from the filestore
// TODO (ramfox): not finished
func (s *fileStore) DeleteAlWorkflows(ctx context.Context) error {
	return fmt.Errorf("not finished: fileStore delete all workflows")
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
		Workflows *Set
	}{
		Workflows: s.workflows,
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.path, data, 0644)
}
