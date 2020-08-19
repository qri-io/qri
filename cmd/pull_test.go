package cmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/registry/regserver"
)

// Test add without any parameters returns an error
func TestPullNoParams(t *testing.T) {
	run := NewTestRunner(t, "test_peer_pull_no_params", "qri_test_add")
	defer run.Delete()

	// add is an old alias for pull, confirm it works by using it:
	err := run.ExecCommand("qri add")
	if err == nil {
		t.Fatal("expected error, did not get one")
	}
	expect := "nothing to pull"
	if expect != err.Error() {
		t.Errorf("expected %q, got %q", expect, err.Error())
	}
}

// Test pull with a temporary registry that we spin up, and push to
func TestPullWithTempRegistry(t *testing.T) {
	run := NewTestRunner(t, "test_peer_pull_with_temp_reg", "qri_test_add")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	reg, cleanup, err := regserver.NewTempRegistry(ctx, "temp_registry", "", run.TestCrypto)
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

	// Save one commit, push, remove locally, pull from the registry
	run.MustExec(t, "qri save me/one_ds --body testdata/movies/body_ten.csv")
	run.MustExec(t, "qri push me/one_ds")
	run.MustExec(t, "qri remove --all me/one_ds")
	run.MustExec(t, "qri pull test_peer_pull_with_temp_reg/one_ds")
	// TODO(dustmop): Actually validate that the command did something, that
	// the dataset was removed from the local repo but now exists again.
}

// Test saving a local dataset, and pulling a foreign dataset, then list the references
func TestPullAndListRefs(t *testing.T) {
	run := NewTestRunnerWithMockRemoteClient(t, "test_peer_pull_and_list", "pull_and_list")
	defer run.Delete()

	// Save a local dataset
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/my_dataset")

	output := run.MustExec(t, "qri list --raw")
	expect := `0 Peername:  test_peer_pull_and_list
  ProfileID: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  Name:      my_dataset
  Path:      /ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF
  FSIPath:   
  Published: false

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Pull a foreign dataset
	run.MustExec(t, "qri pull other_peer/their_dataset")

	output = run.MustExec(t, "qri list --raw")
	expect = `0 Peername:  other_peer
  ProfileID: QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD
  Name:      their_dataset
  Path:      /ipfs/QmQ5292CNJFPsTkodSSwEqgjRdrvBB38k1ZdiUJxvahGgE
  FSIPath:   
  Published: false
1 Peername:  test_peer_pull_and_list
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

// Test adding a foreign dataset, and then getting it
func TestPullAndGet(t *testing.T) {
	run := NewTestRunnerWithMockRemoteClient(t, "test_peer_pull_and_get", "pull_and_get")
	defer run.Delete()

	// Pull a foreign dataset
	run.MustExec(t, "qri pull other_peer/their_dataset")

	output := run.MustExec(t, "qri get other_peer/their_dataset")
	expect := `bodyPath: /ipfs/QmbJWAESqCsf4RFCqEY7jecCashj8usXiyDNfKtZCwwzGb
commit:
  message: created dataset
  path: /ipfs/QmUEtKXFs6Eaz7wxUVpN9riULsnj1hXAoBeHRT79yWL7ze
  qri: cm:0
  signature: ZHoSiqRVGLBKmkRcGaTuaiUGvbaS1Yu+13KtIlYFOnBzDAzZ/pfD1iAykEYp/vMCtKLhFb8s6P7Bnggf2erZQSX5Vd1sQBLKXt7F4fAZ0tS7J5bdalZh4chc6WjvI4VSnk/H4k/ldl5KSYvP3rN7SFY7S/X8zKkirr+aRQRLW+LqcMbYP1h27JsojIM94NIzBBwUkTYLXMaNForx2SxQamWD6Rkcy5Uz82hTjrNVnczJXeCrMR1zyi+LHThoaLDuYfUxIgkprJDrjb0x4fGM3M5DbfuSKlH1iOrXuxzJXDedmEc6Eb48dqgZ/6bpQ8Ij7rc3PtJOu6izLv6MZ3s57g==
  timestamp: "2001-01-01T01:01:01.000000001Z"
  title: created dataset
name: their_dataset
path: /ipfs/QmW8PjK4a3gJgbFr7mHzseCYaPLmhpfphjv5cgXFFDAMtk
peername: other_peer
qri: ds:0
structure:
  checksum: QmSvPd3sHK7iWgZuW47fyLy4CaZQe2DwxvRhrJ39VpBVMK
  depth: 1
  format: json
  length: 2
  qri: st:0
  schema:
    type: object

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

// Test pull a foreign dataset and check it out to a working directory
func TestPullWithCheckout(t *testing.T) {
	run := NewFSITestRunnerWithMockRemoteClient(t, "test_peer_pull_fsi_checkout", "pull_with_checkout")
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
  Path:      /ipfs/QmW8PjK4a3gJgbFr7mHzseCYaPLmhpfphjv5cgXFFDAMtk
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

// TestPullWithDivergentAuthorID tests that logbooks that disagree about their creation
// can be merged and will resolve locally after being merged
func TestPullWithDivergentAuthorID(t *testing.T) {
	// The MockRemoteClient uses peer 1. By using the same peer, we end up with nodes
	// that have the same profileID, but different logbook data.
	testPeerNum := 1
	run := NewTestRunnerUsingPeerInfoWithMockRemoteClient(t, testPeerNum, "test_peer_pull_divergent", "pull_divergent")
	defer run.Delete()

	// Save our dataset
	run.MustExec(t, "qri save test_peer_pull_divergent/one_ds --body testdata/movies/body_ten.csv")

	// Pull a dataset made by the same profileID
	run.MustExec(t, "qri pull test_peer_pull_divergent/two_ds")

	// Get the first dataset
	run.MustExec(t, "qri get me/one_ds")

	// Get the second dataset
	run.MustExec(t, "qri get me/two_ds")
}
