package spec

import (
	"errors"
	"testing"
	"time"

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

	wid := workflow.ID("test id")
	expectedRuns := [3]*run.State{}
	now := time.Now()
	for i := 2; i >= 0; i-- {
		r := &run.State{WorkflowID: wid}
		st := now.Add(time.Duration(0-i) * time.Hour)
		et := st.Add(time.Minute)
		r.StartTime = &st
		r.StopTime = &et
		if i == 0 {
			r.Status = run.RSWaiting
		}
		expectedRun, err := store.Put(r)
		if err != nil {
			t.Fatalf("store.Put unexpected error: %s", err)
		}
		expectedRuns[i] = expectedRun
	}

	if _, err := store.Get("bad run id"); !errors.Is(err, run.ErrNotFound) {
		t.Fatalf("store.Get should emit a run.ErrNotFound error if given an unknown run ID")
	}
	for i := 0; i < 3; i++ {
		got, err := store.Get(expectedRuns[i].ID)
		if err != nil {
			t.Fatalf("store.Get unexpected error: %s", err)
		}

		if diff := cmp.Diff(expectedRuns[i], got); diff != "" {
			t.Errorf("store.Get i=%d mismatch (-want +got):\n%s", i, diff)
		}
	}

	badWID := workflow.ID("bad id")
	if _, err := store.Count(badWID); !errors.Is(err, run.ErrUnknownWorkflowID) {
		t.Fatalf("store.Count should emit a run.ErrUnknownWorkflowID error when given a workflow ID not associated with any runs in the Store")
	}

	gotCount, err := store.Count(wid)
	if err != nil {
		t.Fatalf("store.Count unexpected error: %s", err)
	}
	if gotCount != 3 {
		t.Errorf("store.Count count mismatch, expected 3, got %d", gotCount)
	}

	if _, err := store.List(badWID, -1, 0); !errors.Is(err, run.ErrUnknownWorkflowID) {
		t.Fatalf("store.List should emit a run.ErrUnknownWorkflowID error when given a workflow ID not associated with any runs in the Store")
	}
	gotRuns, err := store.List(wid, -1, 0)
	if err != nil {
		t.Fatalf("store.List unexpected error: %s", err)
	}
	if diff := cmp.Diff(expectedRuns[:], gotRuns); diff != "" {
		t.Errorf("store.List mismatch (-want +got):\n%s", diff)
	}

	if _, err := store.GetLatest(badWID); !errors.Is(err, run.ErrUnknownWorkflowID) {
		t.Fatalf("store.GetLatest should emit a run.ErrUnknownWorkflowID error when given a workflow ID not associated with any runs in the Store")
	}
	got, err := store.GetLatest(wid)
	if err != nil {
		t.Fatalf("store.GetLatest unexpected error: %s", err)
	}
	if diff := cmp.Diff(expectedRuns[0], got); diff != "" {
		t.Errorf("store.GetLatest mismatch (-want +got): \n%s", diff)
	}

	if _, err := store.GetStatus(badWID); !errors.Is(err, run.ErrUnknownWorkflowID) {
		t.Fatalf("store.GetStatus should emit a run.ErrUnknownWorkflowID error when given a workflow ID not associated with any runs in the Store")
	}
	gotStatus, err := store.GetStatus(wid)
	if err != nil {
		t.Fatalf("store.GetStatus unexpected error: %s", err)
	}
	if gotStatus != run.RSWaiting {
		t.Errorf("store.GetStatus mismatch: expected %q, got %q", run.RSWaiting, gotStatus)
	}

	gotRuns, err = store.ListByStatus(run.RSWaiting, -1, 0)
	if err != nil {
		t.Fatalf("store.ListByStatus unexpected error: %s", err)
	}

	if diff := cmp.Diff(expectedRuns[:1], gotRuns); diff != "" {
		t.Errorf("store.ListByStatus mismatch (-want +got): \n%s", diff)
	}
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

	wid := workflow.ID("assert test id")
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
