package workflow

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

var (
	// ErrNotFound indicates that the workflow was not found in the store
	ErrNotFound = fmt.Errorf("workflow not found")
	// ErrWorkflowForDatasetExists indicates that a workflow associated
	// with the given dataset already exists
	ErrWorkflowForDatasetExists = fmt.Errorf("a workflow associated with the given dataset ID already exists")
	// ErrPutDatasetIDMismatch indicates the given workflow's DatasetID does
	// not match the one currently stored
	ErrPutDatasetIDMismatch = fmt.Errorf("the workflow's DatasetID does not match the DatasetID of the associated workflow currently in the store")
	// ErrPutOwnerIDMismatch indicates the given workflow's OwnerID does
	// not match the one currently stored
	ErrPutOwnerIDMismatch = fmt.Errorf("the given workflow's OwnerID does not match the OwnerID of the associated workflow currently in the store")
)

// Store manages & stores workflows, allowing listing and updating of workflows
type Store interface {
	Lister
	Get(wid ID) (*Workflow, error)
	GetByDatasetID(did string) (*Workflow, error)
	Remove(id ID) error
	Put(wf *Workflow) (*Workflow, error)
}

// A Lister lists entries from a workflow store
type Lister interface {
	List(ctx context.Context, limit, offset int) ([]*Workflow, error)
	ListDeployed(ctx context.Context, limit, offset int) ([]*Workflow, error)
}

// MemStore is an in memory representation of a Store
type MemStore struct {
	mu        *sync.Mutex
	workflows map[ID]*Workflow
}

var _ Store = (*MemStore)(nil)

// NewMemStore return a MemStore
func NewMemStore() *MemStore {
	return &MemStore{
		mu:        &sync.Mutex{},
		workflows: map[ID]*Workflow{},
	}
}

func (m *MemStore) Put(wf *Workflow) (*Workflow, error) {
	if wf == nil {
		return nil, ErrNilWorkflow
	}
	if wf.ID != "" {
		fetchedWF, err := m.Get(wf.ID)
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
		if _, err := m.GetByDatasetID(wf.DatasetID); !errors.Is(err, ErrNotFound) {
			return nil, ErrWorkflowForDatasetExists
		}
		wf.ID = NewID()
	}
	if err := wf.Validate(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.workflows[wf.ID] = wf
	m.mu.Unlock()
	return wf, nil
}

// Get fetches a Workflow using the associated ID
func (m *MemStore) Get(wid ID) (*Workflow, error) {
	m.mu.Lock()
	wf, ok := m.workflows[wid]
	m.mu.Unlock()
	if !ok {
		return nil, ErrNotFound
	}
	return wf, nil
}

// GetByDatasetID fetches a workflow using the dataset ID
func (m *MemStore) GetByDatasetID(did string) (*Workflow, error) {
	if did == "" {
		return nil, ErrNotFound
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, wf := range m.workflows {
		if wf.DatasetID == did {
			return wf, nil
		}
	}
	return nil, ErrNotFound
}

// Remove removes a workflow from a store
func (m *MemStore) Remove(id ID) error {
	m.mu.Lock()
	_, ok := m.workflows[id]
	if !ok {
		return ErrNotFound
	}
	delete(m.workflows, id)
	m.mu.Unlock()
	return nil
}

// List lists all the workflows in the store, by decending order from time of
// creation
func (m *MemStore) List(ctx context.Context, limit, offset int) ([]*Workflow, error) {
	wfs := NewWorkflowSet()
	fetchAll := false
	switch {
	case limit == -1 && offset == 0:
		fetchAll = true
	case limit < 0:
		return nil, fmt.Errorf("limit of %d is out of bounds", limit)
	case offset < 0:
		return nil, fmt.Errorf("offset of %d is out of bounds", offset)
	case limit == 0:
		return []*Workflow{}, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, wf := range m.workflows {
		wfs.Add(wf)
	}

	if offset > wfs.Len() {
		return []*Workflow{}, nil
	}

	start := offset
	end := offset + limit
	if end > wfs.Len() || fetchAll {
		end = wfs.Len()
	}

	sort.Sort(wfs)
	return wfs.Slice(start, end), nil
}

// ListDeployed lists all the workflows in the store that are deployed, by
// decending order from time of creation
func (m *MemStore) ListDeployed(ctx context.Context, limit, offset int) ([]*Workflow, error) {
	wfs := NewWorkflowSet()
	fetchAll := false
	switch {
	case limit == -1 && offset == 0:
		fetchAll = true
	case limit < 0:
		return nil, fmt.Errorf("limit of %d is out of bounds", limit)
	case offset < 0:
		return nil, fmt.Errorf("offset of %d is out of bounds", offset)
	case limit == 0:
		return []*Workflow{}, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, wf := range m.workflows {
		if wf.Deployed {
			wfs.Add(wf)
		}
	}

	if offset > wfs.Len() {
		return []*Workflow{}, nil
	}

	start := offset
	end := offset + limit
	if end > wfs.Len() || fetchAll {
		end = wfs.Len()
	}

	sort.Sort(wfs)
	return wfs.Slice(start, end), nil
}
