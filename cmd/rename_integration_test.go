package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	expect = "repo: not found"
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
	}

	// New dataset name can be used, but still has no history
	err = run.ExecCommand("qri get me/remove_second_name")
	if err == nil {
		t.Error("expected error, did not get one")
	}
	expect = "repo: no history"
	if err.Error() != expect {
		t.Errorf("error mismatch, expect: %s, got: %s", expect, err.Error())
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
