package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/qri-io/qri/profile"
)

// Store manages & stores workflows, allowing listing and updating of workflows
type Store interface {
	Lister
	// Create creates a new Workflow and adds it to the Store. If a workflow
	// with that dataset id already exists, it emits an
	// `ErrWorkflowForDatasetExists` error
	Create(did string, pid profile.ID, triggers []Trigger, hooks []Hook) (*Workflow, error)
	Get(wid ID) (*Workflow, error)
	GetDatasetWorkflow(did string) (*Workflow, error)
	Remove(id ID) error
	Update(wf *Workflow) error
	Deploy(id ID) error
	Undeploy(id ID) error
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

// Create creates a new Workflow and adds it to the Store. It does not check that
// the given dataset or peer ids are valid, beyond that they are not empty
// There should only be one workflow per dataset id
func (m *MemStore) Create(did string, pid profile.ID, triggers []Trigger, hooks []Hook) (*Workflow, error) {
	if did == "" {
		return nil, fmt.Errorf("dataset ID required")
	}
	if pid == "" {
		return nil, fmt.Errorf("profile ID required")
	}
	if _, err := m.GetDatasetWorkflow(did); !errors.Is(err, ErrNotFound) {
		return nil, ErrWorkflowForDatasetExists
	}

	now := time.Now()
	wf := &Workflow{
		ID:        NewID(),
		DatasetID: did,
		OwnerID:   pid,
		Created:   &now,
		Triggers:  triggers,
		Hooks:     hooks,
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

// GetDatasetWorkflow fetches a workflow using the dataset ID
func (m *MemStore) GetDatasetWorkflow(did string) (*Workflow, error) {
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

// Update updates a workflow with changes
func (m *MemStore) Update(wf *Workflow) error {
	if wf == nil {
		return ErrNilWorkflow
	}
	if _, err := m.Get(wf.ID); errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	m.mu.Lock()
	m.workflows[wf.ID] = wf
	m.mu.Unlock()
	return nil
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

// Deploy enables the workflow to be run by an orchestrator
func (m *MemStore) Deploy(id ID) error {
	wf, err := m.Get(id)
	if err != nil {
		return err
	}
	wf.Deployed = true
	return m.Update(wf)
}

// Undeploy prevents the workflow from being run by an orchestrator
func (m *MemStore) Undeploy(id ID) error {
	wf, err := m.Get(id)
	if err != nil {
		return err
	}
	wf.Deployed = false
	return m.Update(wf)
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

	if fetchAll {
		limit = wfs.Len()
	}
	if offset > wfs.Len() {
		return nil, fmt.Errorf("offset of %d is out of bounds", offset)
	}
	sort.Sort(wfs)
	if offset+limit > wfs.Len() {
		limit = wfs.Len()
	}
	return wfs.Slice(offset, limit), nil
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

	if fetchAll {
		limit = wfs.Len()
	}
	if offset > wfs.Len() {
		return nil, fmt.Errorf("offset of %d is out of bounds", offset)
	}
	sort.Sort(wfs)
	if offset+limit > wfs.Len() {
		limit = wfs.Len()
	}
	return wfs.Slice(offset, limit), nil
}

// WorkflowSet is a collection of Workflows that implements the sort.Interface,
// sorting a list of WorkflowSet in reverse-chronological-then-alphabetical order
type WorkflowSet struct {
	set []*Workflow
}

// NewWorkflowSet constructs a workflow set.
func NewWorkflowSet() *WorkflowSet {
	return &WorkflowSet{}
}

func (js WorkflowSet) Len() int { return len(js.set) }
func (js WorkflowSet) Less(i, j int) bool {
	return lessNilTime(js.set[i].Created, js.set[j].Created)
}
func (js WorkflowSet) Swap(i, j int) { js.set[i], js.set[j] = js.set[j], js.set[i] }

func (js *WorkflowSet) Add(j *Workflow) {
	if js == nil {
		*js = WorkflowSet{set: []*Workflow{j}}
		return
	}

	for i, workflow := range js.set {
		if workflow.ID == j.ID {
			js.set[i] = j
			return
		}
	}
	js.set = append(js.set, j)
	sort.Sort(js)
}

func (js *WorkflowSet) Remove(id ID) (removed bool) {
	for i, workflow := range js.set {
		if workflow.ID == id {
			if i+1 == len(js.set) {
				js.set = js.set[:i]
				return true
			}

			js.set = append(js.set[:i], js.set[i+1:]...)
			return true
		}
	}
	return false
}

func (js *WorkflowSet) Slice(start, stop int) []*Workflow {
	if start < 0 || stop < 0 {
		return []*Workflow{}
	}
	if stop > js.Len() {
		stop = js.Len()
	}
	return js.set[start:stop]
}

// MarshalJSON serializes WorkflowSet to an array of Workflows
func (js WorkflowSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(js.set)
}

// UnmarshalJSON deserializes from a JSON array
func (js *WorkflowSet) UnmarshalJSON(data []byte) error {
	set := []*Workflow{}
	if err := json.Unmarshal(data, &set); err != nil {
		return err
	}
	js.set = set
	return nil
}

func lessNilTime(a, b *time.Time) bool {
	if a == nil && b != nil {
		return true
	} else if a != nil && b == nil {
		return false
	} else if a == nil && b == nil {
		return false
	}
	return a.After(*b)
}
