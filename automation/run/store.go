package run

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/event"
)

var (
	// ErrNotFound indicates that the run.State was not found in the Store
	ErrNotFound = fmt.Errorf("run not found")
	// ErrUnknownWorkflowID indicates that the given workflow.ID has no
	// associated run.State in the Store
	ErrUnknownWorkflowID = fmt.Errorf("unknown workflow ID")
	// ErrPutWorkflowIDMismatch indicates the given run.ID is associated with
	// a different WorkflowID than the one in the run.State
	ErrPutWorkflowIDMismatch = fmt.Errorf("run.State's WorkflowID does not match the WorkflowID of the associated run.State currently in the store")
)

// Store stores and updates run.States. It is also responsible for tracking the
// number of times a workflow has run, as well as listening for transform
// events and updating the associated the run.State accordingly
type Store interface {
	// Put adds a run state to the Store. If the given RunID is empty, it
	// creates a new run.State. If RunID is not empty, it updates the
	// associated run.
	Put(r *State) (*State, error)
	// Get gets the associated run.State
	Get(id string) (*State, error)
	// Count returns the number of runs for a given workflow.ID
	Count(wid workflow.ID) (int, error)
	// List lists all the runs associated with the workflow.ID in reverse
	// chronological order
	List(wid workflow.ID, lp params.List) ([]*State, error)
	// GetLatest returns the most recent run associated with the workflow id
	GetLatest(wid workflow.ID) (*State, error)
	// GetStatus returns the status of the latest run based on the
	// workflow.ID
	GetStatus(wid workflow.ID) (Status, error)
	// ListByStatus returns a list of run.State entries with a given status
	// looking only at the most recent run of each Workflow
	ListByStatus(s Status, lp params.List) ([]*State, error)
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
	mu              sync.Mutex
	bus             event.Bus
	workflows       map[workflow.ID]*workflowMeta
	runs            map[string]*State
	handleRunEvents event.Handler
}

type workflowMeta struct {
	count  int
	runIDs []string
}

func newWorkflowMeta() *workflowMeta {
	return &workflowMeta{
		count:  0,
		runIDs: []string{},
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

// Put adds a run.State to a MemStore
func (s *MemStore) Put(r *State) (*State, error) {
	if r == nil {
		return nil, fmt.Errorf("run is nil")
	}
	run := r.Copy()
	if run.ID != "" {
		if fetchedR, err := s.Get(run.ID); !errors.Is(err, ErrNotFound) {
			if fetchedR.WorkflowID != run.WorkflowID {
				return nil, ErrPutWorkflowIDMismatch
			}
		}
	} else {
		run.ID = NewID()
	}
	if err := run.Validate(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.workflows[run.WorkflowID]
	if !ok {
		wf := newWorkflowMeta()
		s.workflows[run.WorkflowID] = wf
	}
	s.runs[run.ID] = run
	s.workflows[run.WorkflowID].count++
	runIDs := s.workflows[run.WorkflowID].runIDs
	s.workflows[run.WorkflowID].runIDs = append(runIDs, run.ID)
	return run, nil
}

// Get fetches a run.State using the associated ID
func (s *MemStore) Get(id string) (*State, error) {
	s.mu.Lock()
	r, ok := s.runs[id]
	s.mu.Unlock()
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

// Count returns the number of runs for a given workflow.ID
func (s *MemStore) Count(wid workflow.ID) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wfm, ok := s.workflows[wid]
	if !ok {
		return 0, fmt.Errorf("%w %q", ErrUnknownWorkflowID, wid)
	}
	return wfm.count, nil
}

// List lists all the runs associated with the workflow.ID in reverse
// chronological order
func (s *MemStore) List(wid workflow.ID, lp params.List) ([]*State, error) {
	if err := lp.Validate(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	wfm, ok := s.workflows[wid]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownWorkflowID, wid)
	}
	runIDs := wfm.runIDs
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
func (s *MemStore) GetLatest(wid workflow.ID) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wfm, ok := s.workflows[wid]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownWorkflowID, wid)
	}
	runIDs := wfm.runIDs
	latestRunID := runIDs[len(runIDs)-1]
	run, ok := s.runs[latestRunID]
	if !ok {
		return nil, fmt.Errorf("run %q missing from the store", latestRunID)
	}
	return run, nil
}

// GetStatus returns the status of the latest run based on the
// workflow.ID
func (s *MemStore) GetStatus(wid workflow.ID) (Status, error) {
	run, err := s.GetLatest(wid)
	if err != nil {
		return "", err
	}
	return run.Status, nil
}

// ListByStatus returns a list of run.State entries with a given status
// looking only at the most recent run of each Workflow
func (s *MemStore) ListByStatus(status Status, lp params.List) ([]*State, error) {
	if err := lp.Validate(); err != nil {
		return nil, err
	}

	set := NewSet()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, wfm := range s.workflows {
		runIDs := wfm.runIDs
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

	if _, ok = s.workflows[run.WorkflowID]; !ok {
		wf := newWorkflowMeta()
		s.workflows[run.WorkflowID] = wf
	}
	s.runs[id] = run
	return nil
}
