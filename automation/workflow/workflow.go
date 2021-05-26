package workflow

import (
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/profile"
)

var (
	log = golog.Logger("workflow")
	// ErrNotFound indicates that the workflow was not found in the store
	ErrNotFound = fmt.Errorf("workflow not found")
	// ErrWorkflowForDatasetExists indicates that a workflow associated
	// with the given dataset already exists
	ErrWorkflowForDatasetExists = fmt.Errorf("a workflow associated with the given dataset ID already exists")
	// ErrNilWorkflow indicates that the given workflow is nil
	ErrNilWorkflow = fmt.Errorf("nil workflow")
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
