package cmd

import (
	"context"
	"time"
	"testing"

	"github.com/qri-io/qri/base/dsfs"
	"github.com/spf13/cobra"
)

// LogTestRunner holds test info integration tests
type LogTestRunner struct {
	RepoRoot    *TestRepoRoot
	Context     context.Context
	ContextDone func()
	TsFunc      func() time.Time
	LocOrig     *time.Location
	CmdR        *cobra.Command
}

// newLogTestRunner returns a new FSITestRunner.
func newLogTestRunner(t *testing.T, peerName, testName string) *LogTestRunner {
	root := NewTestRepoRoot(t, peerName, testName)

	run := LogTestRunner{}
	run.RepoRoot = &root
	run.Context, run.ContextDone = context.WithCancel(context.Background())

	// To keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	counter := 0
	run.TsFunc = dsfs.Timestamp
	dsfs.Timestamp = func() time.Time {
		counter++
		return time.Date(2001, 01, 01, 01, 01, counter, 01, time.UTC)
	}

	// Set the location to New York so that timezone printing is consistent
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	run.LocOrig = location
	StringerLocation = location

	return &run
}

// Delete cleans up after a LogTestRunner is done being used.
func (run *LogTestRunner) Delete() {
	dsfs.Timestamp = run.TsFunc
	StringerLocation = run.LocOrig
	run.ContextDone()
	run.RepoRoot.Delete()
}

// ExecCommand executes the given command string
func (run *LogTestRunner) ExecCommand(cmdText string) error {
	run.CmdR = run.RepoRoot.CreateCommandRunner(run.Context)
	return executeCommand(run.CmdR, cmdText)
}

// GetCommandOutput returns the standard output from the previously executed command
func (run *LogTestRunner) GetCommandOutput() string {
	return run.RepoRoot.GetOutput()
}

// Test that deleting an entire dataset works properly with the logbook.
func TestLogAndDeletes(t *testing.T) {
	run := newLogTestRunner(t, "test_peer", "qri_test_log_and_deletes")
	defer run.Delete()

	// Save a dataset containing a body.json, no meta, nothing special.
	err := run.ExecCommand("qri save --body=testdata/movies/body_two.json me/log_test")
	if err != nil {
		t.Fatal(err)
	}

	// Log should have exactly one version
	err = run.ExecCommand("qri log me/log_test")
	if err != nil {
		t.Fatal(err)
	}
	output := run.GetCommandOutput()
	expect := `1   Commit:  /ipfs/QmU8y7DoE5Zp2FcWS7UxmBAYadqkQWvJf9u9FxQjgKoQin
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    79 B

    created dataset

`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri log (-want +got):\n%s", diff)
	}

	// Save anoter dataset version
	err = run.ExecCommand("qri save --body=testdata/movies/body_four.json me/log_test")
	if err != nil {
		t.Fatal(err)
	}

	// Log should have two versions
	err = run.ExecCommand("qri log me/log_test")
	if err != nil {
		t.Fatal(err)
	}
	output = run.GetCommandOutput()
	expect = `1   Commit:  /ipfs/QmUARzGLtSzGSsU6nqQw93Ac57nBQHX1VhmwJzQKBnzkk9
    Date:    Sun Dec 31 20:01:02 EST 2000
    Storage: local
    Size:    137 B

    Structure: 3 changes

    	- modified checksum
    	- modified entries
    	- ...
    ...modified length

2   Commit:  /ipfs/QmU8y7DoE5Zp2FcWS7UxmBAYadqkQWvJf9u9FxQjgKoQin
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    79 B

    created dataset

`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri log (-want +got):\n%s", diff)
	}

	// Save anoter dataset version
	err = run.ExecCommand("qri remove --revisions=1 me/log_test")
	if err != nil {
		t.Fatal(err)
	}

	// Log should only have the first version
	err = run.ExecCommand("qri log me/log_test")
	if err != nil {
		t.Fatal(err)
	}
	output = run.GetCommandOutput()
	expect = `1   Commit:  /ipfs/QmU8y7DoE5Zp2FcWS7UxmBAYadqkQWvJf9u9FxQjgKoQin
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    79 B

    created dataset

`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri log (-want +got):\n%s", diff)
	}

	// TODO(dlong): Get the logbook log, verify that it contains 4 entries (init, 2 saves, delete).
	// Don't render the raw output, just inspect the number of rows, so that we don't couple
	// tightly with the logbook's internal representation.

	// Save anoter dataset version
	err = run.ExecCommand("qri remove --all me/log_test")
	if err != nil {
		t.Fatal(err)
	}

	// Save anoter dataset version
	err = run.ExecCommand("qri save --body=testdata/movies/body_ten.csv me/log_test")
	if err != nil {
		t.Fatal(err)
	}

	// Log should only have the new version
	err = run.ExecCommand("qri log me/log_test")
	if err != nil {
		t.Fatal(err)
	}
	output = run.GetCommandOutput()
	expect = `1   Commit:  /ipfs/QmeEtuVo2fQKCUaVsCmMLsu4eTrHSABKbiUUkuJrEgbv5p
    Date:    Sun Dec 31 20:01:03 EST 2000
    Storage: local
    Size:    224 B

    created dataset

`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri log (-want +got):\n%s", diff)
	}

	// TODO(dlong): Get the logbook, verify that it contains 2 books. The first should end
	// with a deletion, the second should have only two entries (1 init, 1 save).
}
