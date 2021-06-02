package run

import (
	"errors"
	"fmt"
	"sync"

	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/event"
)

var (
	// ErrNilRun indicates that the run.State is nil
	ErrNilRun = fmt.Errorf("nil run")
	// ErrNotFound indicates that the run.State was not found in the Store
	ErrNotFound = fmt.Errorf("run not found")
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
	// Get gets the assocaited run.State
	Get(id string) (*State, error)
	// // Count returns the number of runs of given id
	// Count(id workflow.ID) (int, error)
	// // List lists all the runs associated with the id
	// List(id workflow.ID) ([]*State, error)
	// // GetLatest returns the most recent run associated with the workflow id
	// GetLatest(id workflow.ID) (*State, error)
	// // SubscribeID subscribes to events emitted w/ the given run id. Uses
	// // `AddTransformEvent` to interpret/change the runState, and calls
	// // Update on the Store
	// SubscribeID(id string) error
	// // GetStatus returns the status of the latest run based on the
	// // workflow.ID
	// GetStatus(id workflow.ID) (Status, error)
	// // ListByStatus returns a list of the most recent run.State entries with
	// // a given status.
	// ListByStatus(s Status) []*State
	// Bus returns the bus that the Store subscribes to
	Bus() event.Bus
}

// MemStore is an in memory representation of a Store
type MemStore struct {
	mu        sync.Mutex
	bus       event.Bus
	workflows map[workflow.ID]*workflowInfo
	runs      map[string]*State
}

type workflowInfo struct {
	count  int
	runIDs []string
}

func newWorkflowInfo() *workflowInfo {
	return &workflowInfo{
		count:  0,
		runIDs: []string{},
	}
}

var _ Store = (*MemStore)(nil)

// NewMemStore returns a MemStore
func NewMemStore(bus event.Bus) *MemStore {
	return &MemStore{
		bus:       bus,
		workflows: map[workflow.ID]*workflowInfo{},
		runs:      map[string]*State{},
	}
}

// Put adds a run.State to a MemStore
func (s *MemStore) Put(r *State) (*State, error) {
	if r == nil {
		return nil, ErrNilRun
	}
	run := &State{}
	run.Copy(r)
	if run.ID != "" {
		fetchedR, err := s.Get(run.ID)
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		if fetchedR.WorkflowID != run.WorkflowID {
			return nil, ErrPutWorkflowIDMismatch
		}
	}
	if run.ID == "" {
		run.ID = NewID()
	}
	if err := run.Validate(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.workflows[run.WorkflowID]
	if !ok {
		wf := newWorkflowInfo()
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

// Bus returns the event.Bus
func (s *MemStore) Bus() event.Bus {
	return s.bus
}
