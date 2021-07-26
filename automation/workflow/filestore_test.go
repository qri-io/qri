package workflow_test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/qri/automation/spec"
	"github.com/qri-io/qri/automation/workflow"
)

func TestFileStoreIntegration(t *testing.T) {
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
}
