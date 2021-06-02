package spec

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/workflow"
)

// AssertRunStore confirms the expected behavior of a run.Store interface
// implementation and the behavior of the store.SubscriptionID method
func AssertRunStore(t *testing.T, store run.Store) {
	AssertRunStoreInterface(t, store)
	AssertRunStoreSubscription(t, store)
}

// AssertRunStoreInterface confirms the expected behavior of a run.Store interface
// implementation
func AssertRunStoreInterface(t *testing.T, store run.Store) {
	assertPut(t, store)
	// create workflow.ID
	// create new run
	// create another new run
	// create another new run, with status "Running"
	// check s.Count(wid) == 3
	// get s.List(wid) = 3, same list in correct order
	// GetLatest(wid), same as latest run
	// GetStatus(wid), confirm "Running"
	// ListByStatus("Running"), get list of one
}

// assertPut confirms the expected behavior of the run.Store's Put method
func assertPut(t *testing.T, store run.Store) {
	if _, err := store.Put(nil); !errors.Is(err, run.ErrNilRun) {
		t.Fatal("store.Put is expected to emit a run.ErrNilRun err if you try to add a nil run.State")
	}
	expected := &run.State{}
	if _, err := store.Put(expected); !errors.Is(err, run.ErrNoWorkflowID) {
		t.Fatal("store.Put is expected emit a run.ErrNoWorkflowID error if you try to add a run.State with no workflow.ID")
	}

	wid := workflow.ID("test id")
	expected.WorkflowID = wid
	got, err := store.Put(expected)
	if err != nil {
		t.Fatalf("store.Put unexpected error: %s", err)
	}
	if got.ID == "" {
		t.Fatal("store.Put is expected to fill the run.ID field if given a run.State with an empty ID")
	}
	runID := got.ID
	expected.ID = runID
	expected.Status = run.RSRunning
	got, err = store.Put(expected)
	if err != nil {
		t.Fatalf("store.Put unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("run.State mismatch (-want +got):\n%s", diff)
	}

	expected.ID = ""
	if _, err := store.Put(expected); err != nil {
		t.Fatalf("store.Put should be able to add multiple entries for a single workflow.ID. Unexpected error %s", err)
	}

	expected.ID = runID
	expected.WorkflowID = workflow.ID("new id")
	if _, err = store.Put(expected); !errors.Is(err, run.ErrPutWorkflowIDMismatch) {
		t.Fatal("store.Put is expected to emit a run.ErrPutWorkflowIDMismatch if the WorkflowID of the given run.State does not match the WorkflowID of the run.State stored")
	}
}

// AssertRunStoreSubscription confirms the expected behavior of the run.Store when
// transform events are emitted on the event.Bus
func AssertRunStoreSubscription(t *testing.T, store run.Store) {
	// bus := store.Bus()
	// emit ETTransformStart
	// use s.Get() to check update between each update
	// emit ETTransformStepStart
	// emit ETTransformPrint
	// emit ETTransformError
	// emit ETTransformDatasetPreview
	// emit ETTransformStepStop
	// emit ETTransformStepSkip
	// emit ETTransformStop
}
