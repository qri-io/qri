package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
)

// Test rename works if dataset has no history
func TestRenameNoHistory(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_rename_no_history")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_no_history")

	// Init as a linked directory
	run.MustExec(t, "qri init --name remove_no_history --format csv")

	// Read .qri-ref file, it contains the reference this directory is linked to
	actual := run.MustReadFile(t, filepath.Join(workDir, ".qri-ref"))
	expect := "test_peer/remove_no_history"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("qri list (-want +got):\n%s", diff)
	}

	// Go up one directory
	parentDir := filepath.Dir(workDir)
	os.Chdir(parentDir)

	// Rename
	run.MustExec(t, "qri rename me/remove_no_history me/remove_second_name")

	// Old dataset name can't be used
	err := run.ExecCommand("qri get me/remove_no_history")
	if err == nil {
		t.Error("expected error, did not get one")
	}
	expect = "reference not found"
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
	}

	// New dataset name can be used
	err = run.ExecCommand("qri get me/remove_second_name")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	// Read .qri-ref file, it contains the new reference name
	actual = run.MustReadFile(t, filepath.Join(workDir, ".qri-ref"))
	expect = "test_peer/remove_second_name"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("qri list (-want +got):\n%s", diff)
	}

	// Test that `qri list` will only show the new ref. Still linked to the old directory name.
	output := run.MustExec(t, "qri list")
	expect = `1   test_peer/remove_second_name
    linked: /tmp/remove_no_history
    0 B, 0 entries, 0 errors

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("qri list (-want +got):\n%s", diff)
	}
}

// Test rename updates the qri-ref link
func TestRenameUpdatesLink(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_rename_update_link")
	defer run.Delete()

	workDir := run.CreateAndChdirToWorkDir("remove_update_link")

	// Init as a linked directory
	run.MustExec(t, "qri init --name remove_update_link --format csv")

	// Save a version
	run.MustExec(t, "qri save")

	// Read .qri-ref file, it contains the reference this directory is linked to
	actual := run.MustReadFile(t, filepath.Join(workDir, ".qri-ref"))
	expect := "test_peer/remove_update_link"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("qri list (-want +got):\n%s", diff)
	}

	// Go up one directory
	parentDir := filepath.Dir(workDir)
	os.Chdir(parentDir)

	// Rename
	run.MustExec(t, "qri rename me/remove_update_link me/remove_second_name")

	// Test that `qri list` will only show the new ref. Still linked to the old directory name.
	output := run.MustExec(t, "qri list")
	expect = `1   test_peer/remove_second_name
    linked: /tmp/remove_update_link
    22 B, 2 entries, 0 errors

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("qri list (-want +got):\n%s", diff)
	}

	// Read .qri-ref file, it contains the new dataset reference
	actual = run.MustReadFile(t, filepath.Join(workDir, ".qri-ref"))
	expect = "test_peer/remove_second_name"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("read .qri-ref (-want +got):\n%s", diff)
	}
}

// Test that rename can be used on names with bad upper-case characters, but only to rename them
// to be valid instead
func TestRenameAwayFromBadCase(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "rename_human")
	defer run.Delete()

	// Create a dataset with a valid name
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/first_name")

	// Cannot rename the dataset to a name with bad upper-case characters
	err := run.ExecCommand("qri rename test_peer/first_name test_peer/useUpperCase")
	if err == nil {
		t.Fatal("expected error, did not get one")
	}
	expectErr := `dataset name may not contain any upper-case letters`
	if diff := cmp.Diff(expectErr, err.Error()); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Construct a dataset
	ds := dataset.Dataset{
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[[\"one\",2],[\"three\",4]]")))

	// Add the dataset to the repo directly, which avoids the name validation check.
	ctx := context.Background()
	run.AddDatasetToRefstore(ctx, t, "test_peer/a_New_Dataset", &ds)

	// Cannot rename the dataset to a name with bad upper-case characters still
	err = run.ExecCommand("qri rename test_peer/a_New_Dataset test_peer/useUpperCase")
	if err == nil {
		t.Fatal("expected error, did not get one")
	}
	expectErr = `dataset name may not contain any upper-case letters`
	if diff := cmp.Diff(expectErr, err.Error()); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Okay to rename a name with bad upper-case characters to a new valid name
	err = run.ExecCommand("qri rename test_peer/a_New_Dataset test_peer/a_new_dataset")
	if err != nil {
		t.Errorf("got error: %s", err)
	}
}
