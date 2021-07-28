package cmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset/dstest"
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
	expect := dstest.Template(t, `0 Peername:  test_peer_pull_and_list
  ProfileID: {{ .profileID }}
  Name:      my_dataset
  Path:      {{ .path }}
  FSIPath:   
  Published: false

`, map[string]string{
		"profileID": "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
		"path":      "/ipfs/QmVoTPfveZmw6nVwz48KNPhcAgMdPwr2UWL4fhYd9pr2GM",
	})

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Pull a foreign dataset
	run.MustExec(t, "qri pull other_peer/their_dataset")

	output = run.MustExec(t, "qri list --raw")
	expect = dstest.Template(t, `0 Peername:  other_peer
  ProfileID: {{ .profileID1 }}
  Name:      their_dataset
  Path:      {{ .path1 }}
  FSIPath:   
  Published: false
1 Peername:  test_peer_pull_and_list
  ProfileID: {{ .profileID2 }}
  Name:      my_dataset
  Path:      {{ .path2 }}
  FSIPath:   
  Published: false

`, map[string]string{
		"profileID1": "QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD",
		"profileID2": "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B",
		"path1":      "/ipfs/QmTa8HQ2kisP2enbiyUw3ordSA1WV3ZBQVNUCFBusHENW4",
		"path2":      "/ipfs/QmVoTPfveZmw6nVwz48KNPhcAgMdPwr2UWL4fhYd9pr2GM",
	})

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
	expect := dstest.Template(t, `bodyPath: {{ .bodyPath }}
commit:
  message: created dataset
  path: {{ .commitPath }}
  qri: cm:0
  signature: {{ .signature }}
  timestamp: "2001-01-01T01:01:01.000000001Z"
  title: created dataset
name: their_dataset
path: {{ .path }}
peername: other_peer
qri: ds:0
stats: {{ .statsPath }}
structure:
  checksum: {{ .bodyPath }}
  depth: 1
  format: json
  length: 2
  path: {{ .structurePath }}
  qri: st:0
  schema:
    type: object

`, map[string]string{
		"signature":     "gySMr/FiT+kz0X2ODXCE5APx/BvPvalw4xlbS8TtSWssEoHlAOdrUNKUfU7j6rjyq7sFJ7hrbIVOn87fx+7arYCvrvikRawd2anzIvIruxfBymS6A0HtAGAOEAvpn3XbDykEjqaomTXS1CyR6wQkwNEgbELCIqwda9UV3ulhUtHMrAyMxvnq3NG6J9wyFB13u133aDVEojJ82mEF5DBFB+VBVbw90S4b/5AxLEUFSt/BCtE1O0lKYCt2x0HK+1fhl85oe3fpqLhLk96qCAR/Ngv4bt0E9NjGi2ltuji8gaDICKe5KRaSXjXlMkwbUq6sXEKgqzfxHXoIAUZnZNwnmg==",
		"bodyPath":      "/ipfs/QmbJWAESqCsf4RFCqEY7jecCashj8usXiyDNfKtZCwwzGb",
		"commitPath":    "/ipfs/QmTTPd47BD4EGpCpuvRwTRqDRF84iAuJmfUUGcfEBuF7he",
		"path":          "/ipfs/Qme666Kphnyw8Sf9sjJaEUp1gQ9PodDyZVW878b6pHny9n",
		"structurePath": "/ipfs/QmWoYVZWDdiNauzeP171hKSdo3p2bFaqDcW6cppb9QugUE",
		"statsPath":     "/ipfs/QmQQkQF2KNBZfFiX33jJ9hu6ivfoHrtgcwMRAezS4dcA7c",
	})

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

// Test pull a foreign dataset and check it out to a working directory
func TestPullWithCheckout(t *testing.T) {
	t.Skip("fsi linking is going away")
	run := NewFSITestRunnerWithMockRemoteClient(t, "test_peer_pull_fsi_checkout", "pull_with_checkout")
	defer run.Delete()

	run.ChdirToRoot()

	// Add and checkout another peer's dataset
	run.MustExec(t, "qri pull other_peer/their_dataset --link workdir")
	workDir := filepath.Join(run.RootPath, "workdir")

	// List references, the dataset should be there, and should be checked out
	actual := run.MustExec(t, "qri list --raw")
	expect := dstest.Template(t, `0 Peername:  other_peer
  ProfileID: {{ .profileID }}
  Name:      their_dataset
  Path:      {{ .path1 }}
  FSIPath:   /tmp/workdir
  Published: false

`, map[string]string{
		"profileID": "QmWYgD49r9HnuXEppQEq1a7SUUryja4QNs9E6XCH2PayCD",
		"path1":     "/ipfs/Qme666Kphnyw8Sf9sjJaEUp1gQ9PodDyZVW878b6pHny9n",
	})
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
