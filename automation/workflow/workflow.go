package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/google/uuid"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/automation/hook"
	"github.com/qri-io/qri/automation/trigger"
	"github.com/qri-io/qri/profile"
)

var (
	log = golog.Logger("workflow")
	// ErrNilWorkflow indicates that the given workflow is nil
	ErrNilWorkflow = fmt.Errorf("nil workflow")
	// ErrNoWorkflowID indicates the workflow is invalid because the ID field is empty
	ErrNoWorkflowID = fmt.Errorf("invalid workflow: empty ID")
	// ErrNoDatasetID indicates the workflow is invalid because the DatasetID field is empty
	ErrNoDatasetID = fmt.Errorf("invalid workflow: empty DatasetID")
	// ErrNoOwnerID indicates the workflow is invalid because the OwnerID field is empty
	ErrNoOwnerID = fmt.Errorf("invalid workflow: empty OwnerID")
	// ErrNilCreated indicates the workflow is invalid because the Created field is empty
	ErrNilCreated = fmt.Errorf("invalid workflow: nil Created")
)

// ID is a string identifier for a workflow
type ID string

// NewID creates a new workflow identifier
func NewID() ID {
	return ID(uuid.New().String())
}

// String returns the underlying id string
func (id ID) String() string { return string(id) }

// SetIDRand sets the random reader that NewID uses as a source of random bytes
// passing in nil will default to crypto.Rand. This can be used to make ID
// generation deterministic for tests. eg:
//    myString := "SomeRandomStringThatIsLong-SoYouCanCallItAsMuchAsNeeded..."
//    workflow.SetIDRand(strings.NewReader(myString))
//    a := NewID()
//    workflow.SetIDRand(strings.NewReader(myString))
//    b := NewID()
func SetIDRand(r io.Reader) {
	uuid.SetRand(r)
}

// A Workflow associates automation with a dataset
type Workflow struct {
	ID        ID
	DatasetID string
	OwnerID   profile.ID
	Created   *time.Time
	Deployed  bool
	Triggers  []trigger.Trigger
	Hooks     []hook.Hook
}

// Validate errors if the workflow is not valid
func (w *Workflow) Validate() error {
	if w == nil {
		return ErrNilWorkflow
	}
	if w.ID == "" {
		return ErrNoWorkflowID
	}
	if w.DatasetID == "" {
		return ErrNoDatasetID
	}
	if w.OwnerID == "" {
		return ErrNoOwnerID
	}
	if w.Created == nil {
		return ErrNilCreated
	}
	return nil
}

// Copy returns a shallow copy of the receiver
func (w *Workflow) Copy() *Workflow {
	if w == nil {
		return nil
	}
	workflow := &Workflow{
		ID:        w.ID,
		DatasetID: w.DatasetID,
		OwnerID:   w.OwnerID,
		Created:   w.Created,
		Deployed:  w.Deployed,
		Triggers:  w.Triggers,
		Hooks:     w.Hooks,
	}
	return workflow
}

// Owner returns the owner id
func (w *Workflow) Owner() profile.ID {
	return w.OwnerID
}

// WorkflowID returns the workflow id as a string
func (w *Workflow) WorkflowID() string {
	return w.ID.String()
}

// ActiveTriggers returns a list of triggers that are currently enabled
func (w *Workflow) ActiveTriggers(triggerType string) []trigger.Trigger {
	activeTriggers := []trigger.Trigger{}
	for _, t := range w.Triggers {
		if t.Active() && t.Type() == triggerType {
			activeTriggers = append(activeTriggers, t)
		}
	}
	return activeTriggers
}

// Set is a collection of Workflows that implements the sort.Interface,
// sorting a list of Set in reverse-chronological-then-alphabetical order
type Set struct {
	set []*Workflow
}

// NewSet constructs a workflow set.
func NewSet() *Set {
	return &Set{}
}

// Len part of the `sort.Interface`
func (s Set) Len() int { return len(s.set) }

// Less part of the `sort.Interface`
func (s Set) Less(i, j int) bool {
	return lessNilTime(s.set[i].Created, s.set[j].Created)
}

// Swap is part of the `sort.Interface`
func (s Set) Swap(i, j int) { s.set[i], s.set[j] = s.set[j], s.set[i] }

// Add adds a Workflow to a Set
func (s *Set) Add(j *Workflow) {
	if s == nil {
		*s = Set{set: []*Workflow{j}}
		return
	}

	for i, workflow := range s.set {
		if workflow.ID == j.ID {
			s.set[i] = j
			return
		}
	}
	s.set = append(s.set, j)
	sort.Sort(s)
}

// Remove removes a Workflow from a Set
func (s *Set) Remove(id ID) (removed bool) {
	for i, workflow := range s.set {
		if workflow.ID == id {
			if i+1 == len(s.set) {
				s.set = s.set[:i]
				return true
			}

			s.set = append(s.set[:i], s.set[i+1:]...)
			return true
		}
	}
	return false
}

// Slice returns a slice of Workflows from position `start` to position `end`
func (s *Set) Slice(start, end int) []*Workflow {
	if start < 0 || end < 0 {
		return []*Workflow{}
	}
	if end > s.Len() {
		end = s.Len()
	}
	return s.set[start:end]
}

// MarshalJSON satisfies the `json.Marshaller` interface
func (s Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.set)
}

// UnmarshalJSON satisfies the `json.Unmarshaller` interface
func (s *Set) UnmarshalJSON(data []byte) error {
	set := []*Workflow{}
	if err := json.Unmarshal(data, &set); err != nil {
		return err
	}
	s.set = set
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
