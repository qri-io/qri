package spec

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/params"
)

// AssertRunStore confirms the expected behavior of a run.Store interface
func AssertRunStore(t *testing.T, store run.Store) {
	ctx := context.Background()
	assertCreatePut(ctx, t, store)

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
		expectedRun, err := store.Create(ctx, r)
		if err != nil {
			t.Fatalf("store.Put unexpected error: %s", err)
		}
		expectedRuns[i] = expectedRun
	}

	if _, err := store.Get(ctx, "bad run id"); !errors.Is(err, run.ErrNotFound) {
		t.Fatalf("store.Get should emit a run.ErrNotFound error if given an unknown run ID")
	}
	for i := 0; i < 3; i++ {
		got, err := store.Get(ctx, expectedRuns[i].ID)
		if err != nil {
			t.Fatalf("store.Get unexpected error: %s", err)
		}

		if diff := cmp.Diff(expectedRuns[i], got); diff != "" {
			t.Errorf("store.Get i=%d mismatch (-want +got):\n%s", i, diff)
		}
	}

	badWID := workflow.ID("bad id")
	if _, err := store.Count(ctx, badWID); !errors.Is(err, run.ErrUnknownWorkflowID) {
		t.Fatalf("store.Count should emit a run.ErrUnknownWorkflowID error when given a workflow ID not associated with any runs in the Store")
	}

	gotCount, err := store.Count(ctx, wid)
	if err != nil {
		t.Fatalf("store.Count unexpected error: %s", err)
	}
	if gotCount != 3 {
		t.Errorf("store.Count count mismatch, expected 3, got %d", gotCount)
	}

	if _, err := store.List(ctx, badWID, params.ListAll); !errors.Is(err, run.ErrUnknownWorkflowID) {
		t.Fatalf("store.List should emit a run.ErrUnknownWorkflowID error when given a workflow ID not associated with any runs in the Store")
	}
	gotRuns, err := store.List(ctx, wid, params.ListAll)
	if err != nil {
		t.Fatalf("store.List unexpected error: %s", err)
	}
	if diff := cmp.Diff(expectedRuns[:], gotRuns); diff != "" {
		t.Errorf("store.List mismatch (-want +got):\n%s", diff)
	}

	if _, err := store.GetLatest(ctx, badWID); !errors.Is(err, run.ErrUnknownWorkflowID) {
		t.Fatalf("store.GetLatest should emit a run.ErrUnknownWorkflowID error when given a workflow ID not associated with any runs in the Store")
	}
	got, err := store.GetLatest(ctx, wid)
	if err != nil {
		t.Fatalf("store.GetLatest unexpected error: %s", err)
	}
	if diff := cmp.Diff(expectedRuns[0], got); diff != "" {
		t.Errorf("store.GetLatest mismatch (-want +got): \n%s", diff)
	}

	if _, err := store.GetStatus(ctx, badWID); !errors.Is(err, run.ErrUnknownWorkflowID) {
		t.Fatalf("store.GetStatus should emit a run.ErrUnknownWorkflowID error when given a workflow ID not associated with any runs in the Store")
	}
	gotStatus, err := store.GetStatus(ctx, wid)
	if err != nil {
		t.Fatalf("store.GetStatus unexpected error: %s", err)
	}
	if gotStatus != run.RSWaiting {
		t.Errorf("store.GetStatus mismatch: expected %q, got %q", run.RSWaiting, gotStatus)
	}

	gotRuns, err = store.ListByStatus(ctx, "", run.RSWaiting, params.ListAll)
	if err != nil {
		t.Fatalf("store.ListByStatus unexpected error: %s", err)
	}

	if diff := cmp.Diff(expectedRuns[:1], gotRuns); diff != "" {
		t.Errorf("store.ListByStatus mismatch (-want +got): \n%s", diff)
	}
}

// assertCreatePut confirms the expected behavior of the run.Store's Put method
func assertCreatePut(ctx context.Context, t *testing.T, store run.Store) {
	if _, err := store.Create(ctx, nil); err == nil {
		t.Fatal("store.Create should error when passed nil")
	}
	expected := &run.State{}
	if _, err := store.Create(ctx, expected); !errors.Is(err, run.ErrNoWorkflowID) {
		t.Fatal("store.Create is expected to emit a run.ErrNoWorkflowID error if you try to create a run.State wit no workflow.ID")
	}

	wid := workflow.ID("assert test id")
	expected.WorkflowID = wid

	got, err := store.Create(ctx, expected)
	if err != nil {
		t.Fatalf("store.Create unexpected error: %s", err)
	}
	if got.ID == "" {
		t.Fatal("store.Create is expected to fill the run.ID field if given a run.State with an empty ID")
	}
	if _, err := store.Create(ctx, got); err == nil {
		t.Fatal("store.Create is expected to error if trying to create a run State with a run ID that already exists in the store")
	}

	got, err = store.Create(ctx, expected)
	if err != nil {
		t.Fatalf("store.Create should be able to add multiple entries for a single workflow.ID. Unexpected error %s", err)
	}

	if _, err := store.Put(ctx, nil); err == nil {
		t.Fatal("store.Put should error when passed nil")
	}

	if _, err := store.Put(ctx, &run.State{}); err == nil {
		t.Fatal("store.Put should error when passed a State with no run ID")
	}
	if _, err := store.Put(ctx, &run.State{ID: "test_id"}); err == nil {
		t.Fatal("store.Put should error if you try to add a run.State with no workflow.ID")
	}
	if _, err := store.Put(ctx, &run.State{ID: "test_id", WorkflowID: "test workflow ID"}); err == nil {
		t.Fatal("store.Put should error when passed a new State")
	}
	runID := got.ID
	expected.ID = runID
	expected.Status = run.RSRunning
	got, err = store.Put(ctx, expected)
	if err != nil {
		t.Fatalf("store.Put unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("run.State mismatch (-want +got):\n%s", diff)
	}

	if _, err := store.Put(ctx, expected); err != nil {
		t.Fatalf("store.Put should be able to update the same run multiple times. Unexpected error: %s", err)
	}

	expected.ID = runID
	expected.WorkflowID = workflow.ID("new id")
	if _, err = store.Put(ctx, expected); err == nil {
		t.Fatal("store.Put should error if the WorkflowID of the given run.State does not match the WorkflowID of the run.State stored")
	}
	count, err := store.Count(ctx, got.WorkflowID)
	if err != nil {
		t.Fatalf("store.Count unexpected error %s", err)
	}
	if count != 2 {
		t.Fatalf("store.Create/store.Put error: creating a new run should increment the run count, updating a run using run.Put should NOT increment the run count. Expected count of %d, got %d", 2, count)
	}
}
