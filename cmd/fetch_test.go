package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
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

	RemoteHost := fmt.Sprintf("http://localhost:%d", RemotePort)

	// Assign peer A as a remote for peer B
	cfgCmdText := fmt.Sprintf("qri config set remotes.a_node %s", RemoteHost)
	b.MustExec(t, cfgCmdText)

	// Have peer B fetch from peer A, output correlates to the log from peer A earlier
	actual = b.MustExec(t, "qri fetch peer_a/test_movies --remote a_node")
	expect = `1   Commit:  /ipfs/QmbjY9YG6xKfrPxiXA9eBkJSZiiRRtfKoaS9LSnyVvCAuA
    Date:    Sun Dec 31 20:02:01 EST 2000
    Storage: remote
    Size:    720 B

    structure updated 3 fields

2   Commit:  /ipfs/QmXfgnK7XmyZcRfKrhDysRh5AcHqQntLy98i4joDqopqx6
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: remote
    Size:    224 B

    created dataset

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Regex that replaces the timestamp with just static text
	fixTs := regexp.MustCompile(`"timestamp":"[0-9TZ.:+-]*"`)

	// Verify the logbook on peer B doesn't contain the fetched info
	output := b.MustExec(t, "qri logbook --raw")
	actual = string(fixTs.ReplaceAll([]byte(output), []byte(`"timestamp":timeStampHere`)))
	expect = `[{"ops":[{"type":"init","model":"user","name":"peer_b","authorID":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B","timestamp":timeStampHere}]}]`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	localInst, err := lib.NewInstance(
		ctx,
		b.RepoRoot.QriPath,
		lib.OptStdIOStreams(),
		lib.OptSetIPFSPath(b.RepoRoot.IPFSPath),
	)

	//
	// Validate the outputs of history and fetch
	//

	logHandler := api.NewLogHandlers(remoteServer.Node())

	// Validates output of history for a remote dataset
	actualStatusCode, actualBody := APICall(
		"GET",
		"/history/peer_a/test_movies",
		nil,
		logHandler.LogHandler)
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody := `{"data":[{"username":"peer_a","profileID":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B","name":"test_movies","path":"/ipfs/QmbjY9YG6xKfrPxiXA9eBkJSZiiRRtfKoaS9LSnyVvCAuA","bodySize":720,"commitTime":"2001-01-01T02:02:01.000000001+01:00","commitTitle":"structure updated 3 fields","commitMessage":"structure:\n\tupdated checksum\n\tupdated entries\n\tupdated length"},{"username":"peer_a","profileID":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B","name":"test_movies","path":"/ipfs/QmXfgnK7XmyZcRfKrhDysRh5AcHqQntLy98i4joDqopqx6","bodySize":224,"commitTime":"2001-01-01T02:01:01.000000001+01:00","commitTitle":"created dataset","commitMessage":"created dataset"}],"meta":{"code":200},"pagination":{"nextUrl":"/history/peer_a/test_movies?page=2"}}`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}

	remClientHandler := api.NewRemoteClientHandlers(localInst, false)

	// Validates output of fetch for a remote dataset
	actualStatusCode, actualBody = APICall(
		"POST",
		"/fetch/peer_a/test_movies",
		map[string]string{
			"remote": "a_node",
		},
		remClientHandler.NewFetchHandler("/fetch"))
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	expectBody = `{"data":[{"username":"peer_a","name":"test_movies","path":"/ipfs/QmbjY9YG6xKfrPxiXA9eBkJSZiiRRtfKoaS9LSnyVvCAuA","foreign":true,"bodySize":720,"commitTime":"2001-01-01T02:02:01.000000001+01:00","commitTitle":"structure updated 3 fields"},{"username":"peer_a","name":"test_movies","path":"/ipfs/QmXfgnK7XmyZcRfKrhDysRh5AcHqQntLy98i4joDqopqx6","foreign":true,"bodySize":224,"commitTime":"2001-01-01T02:01:01.000000001+01:00","commitTitle":"created dataset"}],"meta":{"code":200}}`
	if expectBody != actualBody {
		t.Errorf("expected body %s, got %s", expectBody, actualBody)
	}
}

// APICall calls the api and returns the status code and body
func APICall(method, reqURL string, params map[string]string, hf http.HandlerFunc) (int, string) {
	// Add parameters from map
	reqParams := url.Values{}
	if params != nil {
		for key := range params {
			reqParams.Set(key, params[key])
		}
	}
	req := httptest.NewRequest(method, reqURL, strings.NewReader(reqParams.Encode()))
	// Set form-encoded header so server will find the parameters
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(reqParams.Encode())))
	w := httptest.NewRecorder()
	hf(w, req)
	res := w.Result()
	statusCode := res.StatusCode
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	return statusCode, string(bodyBytes)
}
