package workflow

import (
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/qri-io/qri/profile"
)

var (
	// ErrNotFound indicates that the workflow was not found in the store
	ErrNotFound = fmt.Errorf("workflow not found")
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
//    run.SetIDRand(strings.NewReader(myString))
//    a := NewID()
//    run.SetIDRand(strings.NewReader(myString))
//    b := NewID()
func SetIDRand(r io.Reader) {
	uuid.SetRand(r)
}

// A Workflow associates automation with a dataset
type Workflow struct {
	ID        ID
	DatasetID string
	OwnerID   profile.ID
	Deployed  bool
}

// Store manages & stores workflows, allowing listing and updating of workflows
type Store interface {
	// Lister
	Create(did string, pid profile.ID) (*Workflow, error)
	Get(wid ID) (*Workflow, error)
	// Remove
	// Update
	// Deploy
	// Undeploy
}

// A Lister lists entries from a workflow store
type Lister interface {
	// List
	// ListDeployed
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
func (m *MemStore) Create(did string, pid profile.ID) (*Workflow, error) {
	if did == "" {
		return nil, fmt.Errorf("dataset ID required")
	}
	if pid == "" {
		return nil, fmt.Errorf("profile ID required")
	}
	wf := &Workflow{
		ID:        NewID(),
		DatasetID: did,
		OwnerID:   pid,
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
