package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
)

var (
	// ErrNotFound indicates that the run.State was not found in the Store
	ErrNotFound = fmt.Errorf("run not found")
	// ErrUnknownWorkflowID indicates that the given workflow.ID has no
	// associated run.State in the Store
	ErrUnknownWorkflowID = fmt.Errorf("unknown workflow ID")
)

// Store stores and updates run.States. It is also responsible for tracking the
// number of times a workflow has run, as well as listening for transform
// events and updating the associated the run.State accordingly
type Store interface {
	// Create adds a new run State to the Store
	Create(ctx context.Context, r *State) (*State, error)
	// Put puts a run State with an existing run ID into the Store
	Put(ctx context.Context, r *State) (*State, error)
	// Get gets the associated run.State
	Get(ctx context.Context, id string) (*State, error)
	// Count returns the number of runs for a given workflow.ID
	Count(ctx context.Context, wid workflow.ID) (int, error)
	// List lists all the runs associated with the workflow.ID in reverse
	// chronological order
	List(ctx context.Context, wid workflow.ID, lp params.List) ([]*State, error)
	// GetLatest returns the most recent run associated with the workflow id
	GetLatest(ctx context.Context, wid workflow.ID) (*State, error)
	// GetStatus returns the status of the latest run based on the
	// workflow.ID
	GetStatus(ctx context.Context, wid workflow.ID) (Status, error)
	// ListByStatus returns a list of run.State entries with a given status
	// looking only at the most recent run of each Workflow
	ListByStatus(ctx context.Context, owner profile.ID, s Status, lp params.List) ([]*State, error)
	// Shutdown closes the store
	Shutdown() error
}

// EventAdder is an extension interface that optimizes writing an event to a run
// Result should equal to calling:
//   run := store.Get(id)
//   run.AddTransformEvent(e)
//   store.Put(run)
type EventAdder interface {
	Store
	// AddEvent writes an event to the store, attaching it to an existing stored
	// run state
	AddEvent(id string, e event.Event) error
}

// MemStore is an in memory representation of a Store
type MemStore struct {
	mu        sync.Mutex
	workflows map[workflow.ID]*workflowMeta
	runs      map[string]*State
}

type workflowMeta struct {
	Count  int      `json:"count"`
	RunIDs []string `json:"runIDs"`
}

func newWorkflowMeta() *workflowMeta {
	return &workflowMeta{
		Count:  0,
		RunIDs: []string{},
	}
}

var (
	_ Store      = (*MemStore)(nil)
	_ EventAdder = (*MemStore)(nil)
)

// NewMemStore returns a MemStore
func NewMemStore() *MemStore {
	s := &MemStore{
		workflows: map[workflow.ID]*workflowMeta{},
		runs:      map[string]*State{},
	}
	return s
}

// Create adds a new run.State to a MemStore
func (s *MemStore) Create(ctx context.Context, r *State) (*State, error) {
	if r == nil {
		return nil, fmt.Errorf("run is nil")
	}
	run := r.Copy()
	if run.ID == "" {
		run.ID = NewID()
	}
	if _, err := s.Get(ctx, run.ID); !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("run with this ID already exists")
	}
	if err := run.Validate(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.workflows[run.WorkflowID]
	if !ok {
		wfm := newWorkflowMeta()
		s.workflows[run.WorkflowID] = wfm
	}
	s.workflows[run.WorkflowID].Count++
	runIDs := s.workflows[run.WorkflowID].RunIDs
	s.workflows[run.WorkflowID].RunIDs = append(runIDs, run.ID)
	s.runs[run.ID] = run
	return run, nil
}

// Put updates an existing run.State in a MemStore
func (s *MemStore) Put(ctx context.Context, r *State) (*State, error) {
	if r == nil {
		return nil, fmt.Errorf("run is nil")
	}
	run := r.Copy()
	if run.ID == "" {
		return nil, fmt.Errorf("run has empty ID")
	}
	fetchedR, err := s.Get(ctx, run.ID)
	if err != nil {
		return nil, ErrNotFound
	}
	if fetchedR.WorkflowID != run.WorkflowID {
		return nil, fmt.Errorf("run.State's WorkflowID does not match the WorkflowID of the associated run.State currently in the store")
	}
	if err := run.Validate(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.ID] = run
	return run, nil
}

// Get fetches a run.State using the associated ID
func (s *MemStore) Get(ctx context.Context, id string) (*State, error) {
	s.mu.Lock()
	r, ok := s.runs[id]
	s.mu.Unlock()
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

// Count returns the number of runs for a given workflow.ID
func (s *MemStore) Count(ctx context.Context, wid workflow.ID) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wfm, ok := s.workflows[wid]
	if !ok {
		return 0, fmt.Errorf("%w %q", ErrUnknownWorkflowID, wid)
	}
	return wfm.Count, nil
}

// List lists all the runs associated with the workflow.ID in reverse
// chronological order
func (s *MemStore) List(ctx context.Context, wid workflow.ID, lp params.List) ([]*State, error) {
	if err := lp.Validate(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	wfm, ok := s.workflows[wid]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownWorkflowID, wid)
	}
	runIDs := wfm.RunIDs
	runs := []*State{}
	for i := len(runIDs) - 1; i >= 0; i-- {
		id := runIDs[i]
		run, ok := s.runs[id]
		if !ok {
			return nil, fmt.Errorf("run %q missing from the store", id)
		}
		runs = append(runs, run)
	}

	if lp.Offset >= len(runs) {
		return []*State{}, nil
	}

	start := lp.Offset
	end := lp.Offset + lp.Limit
	if end > len(runs) || lp.All() {
		end = len(runs)
	}
	return runs[start:end], nil
}

// GetLatest returns the most recent run associated with the workflow id
func (s *MemStore) GetLatest(ctx context.Context, wid workflow.ID) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wfm, ok := s.workflows[wid]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownWorkflowID, wid)
	}
	runIDs := wfm.RunIDs
	latestRunID := runIDs[len(runIDs)-1]
	run, ok := s.runs[latestRunID]
	if !ok {
		return nil, fmt.Errorf("run %q missing from the store", latestRunID)
	}
	return run, nil
}

// GetStatus returns the status of the latest run based on the
// workflow.ID
func (s *MemStore) GetStatus(ctx context.Context, wid workflow.ID) (Status, error) {
	run, err := s.GetLatest(ctx, wid)
	if err != nil {
		return "", err
	}
	return run.Status, nil
}

// ListByStatus returns a list of run.State entries with a given status
// looking only at the most recent run of each Workflow
func (s *MemStore) ListByStatus(ctx context.Context, owner profile.ID, status Status, lp params.List) ([]*State, error) {
	if err := lp.Validate(); err != nil {
		return nil, err
	}

	set := NewSet()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, wfm := range s.workflows {
		runIDs := wfm.RunIDs
		rid := runIDs[len(runIDs)-1]
		run, ok := s.runs[rid]
		if !ok {
			return nil, fmt.Errorf("run %q missing from the store", rid)
		}
		if run.Status == status {
			set.Add(run)
		}
	}

	if lp.Offset >= set.Len() {
		return []*State{}, nil
	}

	start := lp.Offset
	end := lp.Offset + lp.Limit
	if end > set.Len() || lp.All() {
		end = set.Len()
	}

	sort.Sort(set)
	return set.Slice(start, end), nil
}

// Shutdown closes the store
func (s *MemStore) Shutdown() error {
	return nil
}

// AddEvent writes an event to the store, attaching it to an existing stored
// run state
func (s *MemStore) AddEvent(id string, e event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, ok := s.runs[id]
	if !ok {
		return ErrNotFound
	}
	if err := run.AddTransformEvent(e); err != nil {
		return fmt.Errorf("adding transform event to run: %w", err)
	}

	s.runs[id] = run
	return nil
}

// MarshalJSON satisfies the json.Marshaller interface
func (s *MemStore) MarshalJSON() ([]byte, error) {
	if s == nil {
		s = NewMemStore()
	}
	return json.Marshal(struct {
		Workflows map[workflow.ID]*workflowMeta `json:"workflows"`
		Runs      map[string]*State             `json:"runs"`
	}{Workflows: s.workflows, Runs: s.runs})
}

// UnmarshalJSON satisfies the json.Unmarshaller interface
func (s *MemStore) UnmarshalJSON(p []byte) error {
	v := struct {
		Workflows map[workflow.ID]*workflowMeta `json:"workflows"`
		Runs      map[string]*State             `json:"runs"`
	}{}

	if err := json.Unmarshal(p, &v); err != nil {
		return err
	}
	s.workflows = v.Workflows
	s.runs = v.Runs
	return nil
}
