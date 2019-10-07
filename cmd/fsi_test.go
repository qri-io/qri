package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFSILinkingCommands(t *testing.T) {
	runner := NewFSITestRunner(t, "fsi_commands")
	defer runner.Delete()

	pwd := runner.CreateAndChdirToWorkDir("save_and_unlink")

	// Init as a linked directory
	if err := runner.ExecCommand("qri init --name save_and_unlink --format csv"); err != nil {
		t.Fatal(err.Error())
	}

	// TODO (b5) - get output of qri list, confirm dataset is linked

	// Save a version of the dataset
	if err := runner.ExecCommand("qri save"); err != nil {
		t.Fatal(err.Error())
	}

	// Verify the directory contains the files that we expect, including .qri-ref link file
	dirContents := listDirectory(pwd)
	expectContents := []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}

	// Unlink the dataset
	if err := runner.ExecCommand("qri fsi unlink me/save_and_unlink"); err != nil {
		t.Errorf("unlinking dataset: %s", err.Error())
	}

	// Verify the directory contains the files that we expect
	dirContents = listDirectory(pwd)
	expectContents = []string{"body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents after unlinking (-want +got):\n%s", diff)
	}

	// TODO (b5) - get output of qri list, confirm dataset is unlinked

	// Link the dataset to the pwd
	if err := runner.ExecCommand("qri fsi link me/save_and_unlink ."); err != nil {
		t.Errorf("unlinking dataset: %s", err.Error())
	}

	dirContents = listDirectory(pwd)
	expectContents = []string{".qri-ref", "body.csv", "meta.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}
}
