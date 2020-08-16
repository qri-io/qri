package cmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/registry/regserver"
	repotest "github.com/qri-io/qri/repo/test"
)

// Test add without any parameters returns an error
func TestPull(t *testing.T) {
	run := NewTestRunner(t, "test_peer_add", "qri_test_add")
	defer run.Delete()

	t.Run("no_params", func(t *testing.T) {
		// add is an old alias for pull, confirm it works by using it:
		err := run.ExecCommand("qri add")
		if err == nil {
			t.Fatal("expected error, did not get one")
		}
		expect := "nothing to pull"
		if expect != err.Error() {
			t.Errorf("expected %q, got %q", expect, err.Error())
		}
	})

	t.Run("create_push_delete_add", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// TODO(dustmop): Move into test runner
		reg, cleanup, err := regserver.NewTempRegistry(ctx, "temp_registry", "", repotest.NewTestCrypto())
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup()
		// TODO(b5): need to defer in this order. the deferred cleanup command blocks on done,
		// which is in turn blocked on cancel. deferring in the other order deadlocks.
		// the smarter way to deal with this is to refactor TempRegistry to use the Done pattern
		defer cancel()

		// Create a mock registry, point our test runner to its URL
		_, httpServer := regserver.NewMockServerRegistry(*reg)
		run.RepoRoot.GetConfig().Registry.Location = httpServer.URL
		err = run.RepoRoot.WriteConfigFile()
		if err != nil {
			t.Fatal(err)
		}

		golog.SetLogLevel("remote", "error")

		// Save one commit
		run.MustExec(t, "qri save me/one_ds --body testdata/movies/body_ten.csv")
		run.MustExec(t, "qri push me/one_ds")
		run.MustExec(t, "qri remove --all me/one_ds")
		run.MustExec(t, "qri pull test_peer_add/one_ds")
	})
}

// Test adding a local dataset, and a foreign dataset, then list the references
func TestPullAndListRefs(t *testing.T) {
	run := NewTestRunnerWithMockRemoteClient(t, "test_peer_add_and_list", "add_and_list")
	defer run.Delete()

	// Save a local dataset
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/my_dataset")

	output := run.MustExec(t, "qri list --raw")
	expect := `0 Peername:  test_peer_add_and_list
  ProfileID: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  Name:      my_dataset
  Path:      /ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF
  FSIPath:   
  Published: false

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Save a foreign dataset
	run.MustExec(t, "qri pull other_peer/their_dataset")

	output = run.MustExec(t, "qri list --raw")
	expect = `0 Peername:  other_peer
  ProfileID: QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
  Name:      their_dataset
  Path:      /ipfs/QmeD7XLpUoz6EKzBBGHQ4dMEsA8veRJDz4Ky2WAjkBM5kt
  FSIPath:   
  Published: false
1 Peername:  test_peer_add_and_list
  ProfileID: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  Name:      my_dataset
  Path:      /ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF
  FSIPath:   
  Published: false

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

// Test pull a foreign dataset and checking it out
func TestPullWithCheckout(t *testing.T) {
	msg := `skipping add with checkout b/c remote.MockClient doesn't implement CloneLogs,
which this test needs. The proper solution is to remove remote.MockClient in 
favour of a setup closer to lib.TwoActorRegistryIntegrationTest`
	t.Skip(msg)
	run := NewFSITestRunnerWithMockRemoteClient(t, "test_peer_add_fsi_checkout", "add_fsi_checkout")
	defer run.Delete()

	run.ChdirToRoot()

	// Add and checkout another peer's dataset
	run.MustExec(t, "qri pull other_peer/their_dataset --link workdir")
	workDir := filepath.Join(run.RootPath, "workdir")

	// List references, the dataset should be there, and should be checked out
	actual := run.MustExec(t, "qri list --raw")
	expect := `0 Peername:  other_peer
  ProfileID: QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
  Name:      their_dataset
  Path:      /ipfs/QmbCV8415B6uM4A1UC6YpPCzGpUd5Kx4txWTcPLoXjEcQP
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
