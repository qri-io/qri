package run_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/spec"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/event"
)

func TestFileStore(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	store, err := run.NewFileStore(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	spec.AssertRunStore(t, store)

	counter := 0
	NowFunc := func() *time.Time {
		t := time.Unix(int64(1234000+counter), 0)
		counter++
		return &t
	}
	r1 := &run.State{
		ID:         "run1",
		WorkflowID: "workflow1",
		Number:     0,
		Status:     "success",
		Message:    "message",
		StartTime:  NowFunc(),
		StopTime:   NowFunc(),
		Duration:   1,
		Steps: []*run.StepState{
			{

				Name:      "download",
				Category:  "download",
				Status:    "success",
				StartTime: NowFunc(),
				StopTime:  NowFunc(),
				Duration:  1,
				Output: []event.Event{
					{
						Type:      event.ETTransformStart,
						Timestamp: 1,
						SessionID: "sessionID",
						Payload: event.TransformLifecycle{
							StepCount: 1,
							Status:    "success",
						},
					},
					{
						Type:      event.ETTransformStepStart,
						Timestamp: 2,
						Payload: event.TransformStepLifecycle{
							Name:     "download",
							Category: "download",
							Status:   "error",
						},
					},
					{
						Type:      event.ETTransformPrint,
						Timestamp: 3,
						Payload:   "printing!",
					},
				},
			},
		},
	}
	type WorkflowMeta struct {
		Count  int      `json:"count"`
		RunIDs []string `json:"runIDs"`
	}
	wfm := &WorkflowMeta{
		Count:  1,
		RunIDs: []string{"run1"},
	}
	mockStore := struct {
		Workflows map[workflow.ID]*WorkflowMeta `json:"workflows"`
		Runs      map[string]*run.State         `json:"runs"`
	}{
		Workflows: map[workflow.ID]*WorkflowMeta{
			workflow.ID("workflow1"): wfm,
		},
		Runs: map[string]*run.State{
			"run1": r1,
		},
	}
	storeBytes, err := json.Marshal(mockStore)
	if err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(tmpdir, "runs.json"), storeBytes, 0644); err != nil {
		t.Fatal(err)
	}

	store, err = run.NewFileStore(tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	gotRun, err := store.Get("run1")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(gotRun, r1); diff != "" {
		t.Errorf("run mismatch (-want +got):\n%s", diff)
	}

	r2 := &run.State{
		ID:         "run2",
		WorkflowID: "workflow1",
		Number:     0,
		Status:     "success",
		Message:    "message",
		StartTime:  NowFunc(),
		StopTime:   NowFunc(),
		Duration:   1,
		Steps: []*run.StepState{
			{

				Name:      "download",
				Category:  "download",
				Status:    "success",
				StartTime: NowFunc(),
				StopTime:  NowFunc(),
				Duration:  1,
				Output: []event.Event{
					{
						Type:      event.ETTransformStart,
						Timestamp: 1,
						SessionID: "sessionID",
						Payload: event.TransformLifecycle{
							StepCount: 1,
							Status:    "success",
						},
					},
					{
						Type:      event.ETTransformStepStart,
						Timestamp: 2,
						Payload: event.TransformStepLifecycle{
							Name:     "download",
							Category: "download",
							Status:   "error",
						},
					},
					{
						Type:      event.ETTransformPrint,
						Timestamp: 3,
						Payload:   "printing!",
					},
				},
			},
		},
	}

	if _, err := store.Create(r2); err != nil {
		t.Fatal(err)
	}

	if err := store.Shutdown(); err != nil {
		t.Fatal(err)
	}

	wfm.Count = 2
	wfm.RunIDs = []string{"run1", "run2"}
	mockStore.Workflows["workflow1"] = wfm
	mockStore.Runs["run2"] = r2

	storeBytes, err = json.Marshal(mockStore)
	if err != nil {
		t.Fatal(err)
	}

	gotBytes, err := ioutil.ReadFile(filepath.Join(tmpdir, "runs.json"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(storeBytes), string(gotBytes)); diff != "" {
		t.Errorf("file mismatch (-want +got):\n%s", diff)
	}
}
