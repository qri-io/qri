package cmd

import (
	"testing"
	"time"
)

// LogTestRunner holds test info integration tests
type LogTestRunner struct {
	TestRunner
	LocOrig *time.Location
}

// newLogTestRunner returns a new FSITestRunner.
func newLogTestRunner(t *testing.T, peerName, testName string) *LogTestRunner {
	run := LogTestRunner{
		TestRunner: *NewTestRunner(t, peerName, testName),
	}

	// Set the location to New York so that timezone printing is consistent
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	run.LocOrig = location
	StringerLocation = location

	// Restore the location function
	run.Teardown = func() {
		StringerLocation = run.LocOrig
	}

	return &run
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
	expect = `1   Commit:  /ipfs/QmNtWUnTsPp6W3c3cFUYLScTHesrHsj6L8qe8LzX1Yqwop
    Date:    Sun Dec 31 20:02:01 EST 2000
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
	expect = `1   Commit:  /ipfs/QmTaaHHC9kMvCtwmDr8ahT4Y3d7yLixdu3AUjsQxsAmHes
    Date:    Sun Dec 31 20:03:01 EST 2000
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
