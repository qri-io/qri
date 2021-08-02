package workflow_test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/automation/spec"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/profile"
)

func TestFileStoreIntegration(t *testing.T) {
	ctx := context.Background()
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	store, err := workflow.NewFileStore(tmpdir)
	if err != nil {
		t.Fatalf("NewFileStore unexpected error: %s", err)
	}
	spec.AssertWorkflowStore(t, store)

	err = os.Remove(filepath.Join(tmpdir, "workflows.json"))
	if err != nil {
		t.Fatalf("removing workflow.json file error: %s", err)
	}

	store, err = workflow.NewFileStore(tmpdir)
	if err != nil {
		t.Fatalf("NewFileStore unexpected error: %s", err)
	}
	spec.AssertWorkflowLister(t, store)

	timestamp := time.Unix(0, 123400000)
	expectedWF1 := &workflow.Workflow{
		ID:      "workflow1",
		InitID:  "dataset1",
		OwnerID: profile.IDB58MustDecode("QmTwtwLMKHHKCrugNxyAaZ31nhBqRUQVysT2xK911n4m6F"),
		Created: &timestamp,
		Active:  true,
		Triggers: []map[string]interface{}{
			{"id": "trigger1"},
		},
		Hooks: []map[string]interface{}{
			{"id": "hook1"},
		},
	}
	set := struct {
		Workflows []*workflow.Workflow `json:"workflows"`
	}{
		Workflows: []*workflow.Workflow{expectedWF1},
	}

	// write workflow store json
	wfsBytes, err := json.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(tmpdir, "workflows.json"), wfsBytes, 0644); err != nil {
		t.Fatal(err)
	}

	store, err = workflow.NewFileStore(tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(ctx, expectedWF1.ID)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expectedWF1, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	// add to it
	expectedWF2 := &workflow.Workflow{
		ID:      "workflow2",
		InitID:  "dataset2",
		OwnerID: profile.IDB58MustDecode("QmTwtwLMKHHKCrugNxyAaZ31nhBqRUQVysT2xK911n4m6F"),
		Created: &timestamp,
		Active:  false,
		Triggers: []map[string]interface{}{
			{"id": "trigger2"},
		},
		Hooks: []map[string]interface{}{
			{"id": "hook2"},
		},
	}
	set.Workflows = []*workflow.Workflow{expectedWF1, expectedWF2}
	wfsBytes, err = json.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Put(ctx, expectedWF2); err != nil {
		t.Fatal(err)
	}
	if err := store.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	gotBytes, err := ioutil.ReadFile(filepath.Join(tmpdir, "workflows.json"))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(wfsBytes), string(gotBytes)); diff != "" {
		t.Errorf("file mismatch (-want +got):\n%s", diff)
	}
}
