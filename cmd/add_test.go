package cmd

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Test add without any parameters returns an error
func TestAddNoParam(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "no_param")
	defer run.Delete()

	err := run.ExecCommand("qri add")
	if err == nil {
		t.Fatal("expected error, did not get one")
	}
	expect := "nothing to add"
	if expect != err.Error() {
		t.Errorf("expected %q, got %q", expect, err.Error())
	}
}

// Test adding a local dataset, and a foreign dataset, then list the references
func TestAddAndListRefs(t *testing.T) {
	run := NewTestRunnerWithMockRemoteClient(t, "test_peer", "add_and_list")
	defer run.Delete()

	// Save a local dataset
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/my_dataset")

	output := run.MustExec(t, "qri list --raw")
	expect := `0 Peername:  test_peer
  ProfileID: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  Name:      my_dataset
  Path:      /ipfs/QmXZnsLPRy9i3xFH2dzHkWG1Pkbs8AWqdhTHCYLCX76BjT
  FSIPath:   
  Published: false

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Save a foreign dataset
	run.MustExec(t, "qri add other_peer/their_dataset")

	output = run.MustExec(t, "qri list --raw")
	expect = `0 Peername:  other_peer
  ProfileID: QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
  Name:      their_dataset
  Path:      /ipfs/QmeQC8QdAVtruVyEyAejZBHFDBhqy99pDhDpdPXkALfhfD
  FSIPath:   
  Published: false
1 Peername:  test_peer
  ProfileID: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  Name:      my_dataset
  Path:      /ipfs/QmXZnsLPRy9i3xFH2dzHkWG1Pkbs8AWqdhTHCYLCX76BjT
  FSIPath:   
  Published: false

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

// Test adding a foreign dataset and checking it out
func TestAddWithCheckout(t *testing.T) {
	run := NewFSITestRunnerWithMockRemoteClient(t, "add_fsi_checkout")
	defer run.Delete()

	run.ChdirToRoot()

	// Add and checkout another peer's dataset
	run.MustExec(t, "qri add other_peer/their_dataset --link workdir")
	workDir := filepath.Join(run.RootPath, "workdir")

	// List references, the dataset should be there, and should be checked out
	actual := run.MustExec(t, "qri list --raw")
	expect := `0 Peername:  other_peer
  ProfileID: QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
  Name:      their_dataset
  Path:      /ipfs/QmeD7XLpUoz6EKzBBGHQ4dMEsA8veRJDz4Ky2WAjkBM5kt
  FSIPath:   /tmp/workdir
  Published: false

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Verify the directory contains the files that we expect.
	dirContents := listDirectory(workDir)
	expectContents := []string{".qri-ref", "body.json", "structure.json"}
	if diff := cmp.Diff(expectContents, dirContents); diff != "" {
		t.Errorf("directory contents (-want +got):\n%s", diff)
	}
}
