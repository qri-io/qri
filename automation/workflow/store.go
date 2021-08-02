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
)

// Store manages & stores workflows, allowing listing and updating of workflows
type Store interface {
	Lister
	// Get fetches a Workflow from the Store using the workflow.ID
	Get(ctx context.Context, wid ID) (*Workflow, error)
	// GetByInitID fetches a Workflow from the Store using the dataset.ID
	GetByInitID(ctx context.Context, initID string) (*Workflow, error)
	// Remove removes a Workflow from the Store using the workflow.ID
	Remove(ctx context.Context, id ID) error
	// Put adds a Workflow to the Store. If there is no ID in the Workflow,
	// Put will create a new ID, record the time in the `Created` field
	// and put the workflow in the store, ensuring that the associated
	// Workflow.InitID is unique. If there is an existing ID, Put will
	// update the entry in the Store, if the given workflow is valid
	Put(ctx context.Context, wf *Workflow) (*Workflow, error)
	// Shutdown closes the store
	Shutdown(ctx context.Context) error
}

// A Lister lists entries from a workflow store
type Lister interface {
	// List lists the Workflows in the Store in reverse chronological order
	// by Workflow.Created time
	List(ctx context.Context, limit, offset int) ([]*Workflow, error)
	// ListDeployed lists the deployed Workflows in the Store in reverse
	// chronological order by Workflow.Created time
	ListDeployed(ctx context.Context, limit, offset int) ([]*Workflow, error)
}

// MemStore is an in memory representation of a Store
type MemStore struct {
	mu        *sync.Mutex
	workflows map[ID]*Workflow
}

var _ Store = (*MemStore)(nil)

// NewMemStore returns a MemStore
func NewMemStore() *MemStore {
	return &MemStore{
		mu:        &sync.Mutex{},
		workflows: map[ID]*Workflow{},
	}
}

// Put adds a Workflow to a MemStore
func (m *MemStore) Put(ctx context.Context, wf *Workflow) (*Workflow, error) {
	if wf == nil {
		return nil, ErrNilWorkflow
	}
	w := wf.Copy()
	if w.ID == "" {
		if _, err := m.GetByInitID(ctx, w.InitID); !errors.Is(err, ErrNotFound) {
			return nil, ErrWorkflowForDatasetExists
		}
		w.ID = NewID()
	}
	if err := w.Validate(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.workflows[w.ID] = w
	m.mu.Unlock()
	return w, nil
}

// Get fetches a Workflow using the associated ID
func (m *MemStore) Get(ctx context.Context, wid ID) (*Workflow, error) {
	m.mu.Lock()
	wf, ok := m.workflows[wid]
	m.mu.Unlock()
	if !ok {
		return nil, ErrNotFound
	}
	return wf, nil
}

// GetByInitID fetches a Workflow using the dataset ID
func (m *MemStore) GetByInitID(ctx context.Context, initID string) (*Workflow, error) {
	if initID == "" {
		return nil, ErrNotFound
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, wf := range m.workflows {
		if wf.InitID == initID {
			return wf, nil
		}
	}
	return nil, ErrNotFound
}

// Remove removes a Workflow from a Store
func (m *MemStore) Remove(ctx context.Context, id ID) error {
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
	wfs := NewSet()
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

	if offset >= wfs.Len() {
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
	wfs := NewSet()
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
		if wf.Active {
			wfs.Add(wf)
		}
	}

	if offset >= wfs.Len() {
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

// Shutdown closes the store
func (m *MemStore) Shutdown(ctx context.Context) error {
	return nil
}
