package cmd

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/api"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/logbook"
)

func TestFetchCommand(t *testing.T) {
	a := NewTestRunner(t, "peer_a", "qri_test_fetch_a")
	defer a.Delete()

	// TODO(dustmop): Move most of the below hooks into a common testRunner. Maybe the basic
	// TestRunner will work?

	// Set the location to New York so that timezone printing is consistent
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	locOrig := StringerLocation
	StringerLocation = location

	// Restore the location function
	a.Teardown = func() {
		StringerLocation = locOrig
	}

	// Hook timestamp generation.
	prevTimestampFunc := logbook.NewTimestamp
	logbook.NewTimestamp = func() int64 {
		return 1000
	}
	defer func() {
		logbook.NewTimestamp = prevTimestampFunc
	}()

	// Save a version with some rows in its body
	a.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")

	// Save another version with more rows
	a.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")

	// Get the log, should have two versions.
	actual := a.MustExec(t, "qri log peer_a/test_movies")
	expect := `1   Commit:  /ipfs/QmbjY9YG6xKfrPxiXA9eBkJSZiiRRtfKoaS9LSnyVvCAuA
    Date:    Sun Dec 31 20:02:01 EST 2000
    Storage: local
    Size:    720 B

    structure updated 3 fields
    structure:
    	updated checksum
    	updated entries
    	updated length

2   Commit:  /ipfs/QmXfgnK7XmyZcRfKrhDysRh5AcHqQntLy98i4joDqopqx6
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    224 B

    created dataset

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Enable remote and RPC in the config
	a.MustExec(t, "qri config set remote.enabled true rpc.enabled false")

	ctx := context.Background()

	// Create a remote that makes these versions available
	remoteInst, err := lib.NewInstance(
		ctx,
		a.RepoRoot.QriPath,
		lib.OptStdIOStreams(),
		lib.OptSetIPFSPath(a.RepoRoot.IPFSPath),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := remoteInst.Connect(ctx); err != nil {
		t.Fatal(err)
	}

	// Made an HTTP server for our remote
	remoteServer := api.New(remoteInst)
	httpServer := &http.Server{}
	httpServer.Handler = api.NewServerRoutes(remoteServer)

	// Serve on an available port
	// TODO(dustmop): This port could actually be randomized to make this more robust
	const RemotePort = 9876
	apiConfig := config.API{
		Enabled:    true,
		Port:       RemotePort,
		RemoteMode: true,
	}
	go api.StartServer(&apiConfig, httpServer)
	defer httpServer.Close()

	// Construct a second peer B.
	b := NewTestRunner(t, "peer_b", "qri_test_fetch_b")
	defer b.Delete()

	// Expect an error when trying to list an unavailable dataset
	err = b.ExecCommand("qri log peer_b/test_movies")
	expectErr := `repo: not found`
	if err == nil {
		t.Fatal("expected fetch on non-existant log to error")
	}
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}

	// Assign peer A as a remote for peer B
	cfgCmdText := fmt.Sprintf("qri config set remotes.a_node http://localhost:%d", RemotePort)
	b.MustExec(t, cfgCmdText)

	// Have peer B fetch from peer A, output correlates to the log from peer A earlier
	actual = b.MustExec(t, "qri fetch peer_a/test_movies --remote a_node")
	expect = `1   peer_a/test_movies
    /ipfs/QmbjY9YG6xKfrPxiXA9eBkJSZiiRRtfKoaS9LSnyVvCAuA
    foreign
    720 B, 0 entries, 0 errors

2   peer_a/test_movies
    /ipfs/QmXfgnK7XmyZcRfKrhDysRh5AcHqQntLy98i4joDqopqx6
    foreign
    224 B, 0 entries, 0 errors

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Regex that replaces the timestamp with just static text
	fixTs := regexp.MustCompile(`"timestamp":"[0-9TZ.:-]*"`)

	// Verify the logbook on peer B doesn't contain the fetched info
	output := b.MustExec(t, "qri logbook --raw")
	actual = string(fixTs.ReplaceAll([]byte(output), []byte(`"timestamp":timeStampHere`)))
	expect = `[{"ops":[{"type":"init","model":"user","name":"peer_b","authorID":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B","timestamp":timeStampHere}]}]`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}
}
