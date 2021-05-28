package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/google/uuid"
	golog "github.com/ipfs/go-log"
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
	// ErrTriggerIDMismatch indicates the workflow id stored in the trigger does not match the workflow id
	ErrTriggerIDMismatch = fmt.Errorf("invalid workflow: trigger ID mismatch")
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
	Triggers  []Trigger
	Hooks     []Hook
}

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

func (js *WorkflowSet) Slice(start, end int) []*Workflow {
	if start < 0 || end < 0 {
		return []*Workflow{}
	}
	if end > js.Len() {
		end = js.Len()
	}
	return js.set[start:end]
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
