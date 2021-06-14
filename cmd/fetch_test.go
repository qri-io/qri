package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/api"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
)

func TestFetchCommand(t *testing.T) {
	t.Skip(`This currently won't work b/c commands can't be executed while the "temp registry" is running.
	New rules state there can only be one instance at a time, and in this case an
	instance exists to back the testRunner. requests to execute commands below don't
	work b/c the instance contends for the repo lock. At least I (b5) think that's
	what's going on :/`)

	a := NewTestRunner(t, "peer_a", "qri_test_fetch_a")
	defer a.Delete()

	// Save a version with some rows in its body
	a.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")

	// Save another version with more rows
	a.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")

	// Get the log, should have two versions.
	actual := a.MustExec(t, "qri log peer_a/test_movies")
	expect := `1   Commit:  /ipfs/QmXmnH1tFKyG493wsFiisZ14N4cjrymZZ6pqK3Vr9vWS2p
    Date:    Sun Dec 31 20:02:01 EST 2000
    Storage: local
    Size:    720 B

    body changed by 70%
    body:
    	changed by 70%

2   Commit:  /ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    224 B

    created dataset from body_ten.csv

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Enable remote and RPC in the config
	a.MustExec(t, "qri config set remote.enabled true rpc.enabled false")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a remote that makes these versions available
	remoteInst, err := lib.NewInstance(
		ctx,
		a.RepoRoot.RootPath,
		lib.OptStdIOStreams(),
		lib.OptSetIPFSPath(a.RepoRoot.IPFSPath),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := remoteInst.ConnectP2P(ctx); err != nil {
		t.Fatal(err)
	}

	// Made an HTTP server for our remote
	remoteServer := api.New(remoteInst)
	httpServer := &http.Server{}
	httpServer.Handler = api.NewServerRoutes(remoteServer)

	// Serve on an available port
	// TODO(dustmop): This port could actually be randomized to make this more robust
	const RemotePort = "9876"
	apiConfig := config.API{
		Enabled: true,
		Address: fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", RemotePort),
	}
	go api.StartServer(&apiConfig, httpServer)
	defer httpServer.Close()

	// Construct a second peer B.
	b := NewTestRunnerWithTempRegistry(t, "peer_b", "qri_test_fetch_b")
	defer b.Delete()

	// Expect an error when trying to list an unavailable dataset
	err = b.ExecCommand("qri log peer_b/test_movies")
	expectErr := `reference not found`
	if err == nil {
		t.Fatal("expected fetch on non-existent log to error")
	}
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}

	RemoteHost := fmt.Sprintf("http://localhost:%s", RemotePort)

	// Assign peer A as a remote for peer B
	cfgCmdText := fmt.Sprintf("qri config set remotes.a_node %s", RemoteHost)
	b.MustExec(t, cfgCmdText)

	// Have peer B fetch from peer A, output correlates to the log from peer A earlier
	actual = b.MustExec(t, "qri log peer_a/test_movies --remote a_node")
	expect = `1   Commit:  /ipfs/QmXmnH1tFKyG493wsFiisZ14N4cjrymZZ6pqK3Vr9vWS2p
    Date:    Sun Dec 31 20:02:01 EST 2000
    Storage: remote
    Size:    720 B

    body changed by 70%

2   Commit:  /ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: remote
    Size:    224 B

    created dataset from body_ten.csv

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// Regex that replaces the timestamp with just static text
	fixTs := regexp.MustCompile(`"(timestamp|commitTime)":\s?"[0-9TZ.:+-]*?"`)

	// Verify the logbook on peer B doesn't contain the fetched info
	output := b.MustExec(t, "qri logbook --raw")
	actual = string(fixTs.ReplaceAll([]byte(output), []byte(`"timestamp":"timeStampHere"`)))
	expect = `[{"ops":[{"type":"init","model":"user","name":"peer_b","authorID":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B","timestamp":"timeStampHere"}]}]`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):%s\n", diff)
	}

	// TODO(dustmop): Try to add the below to a separate test in api/. Need to populate the peers
	// in a fashion similar to api/fsi_test.go's `TestNoHistory`.

	localInst, err := lib.NewInstance(
		ctx,
		b.RepoRoot.RootPath,
		lib.OptStdIOStreams(),
		lib.OptSetIPFSPath(b.RepoRoot.IPFSPath),
	)
	localLogHandler := lib.NewHTTPRequestHandler(localInst, "log.history")

	//
	// Validate the outputs of history and fetch
	//

	remoteLogHandler := lib.NewHTTPRequestHandler(remoteServer.Instance, "log.history")

	// Validates output of history for a remote dataset getting history for a
	// dataset in its own namespace in its own repo
	actualStatusCode, actualBody := APICall(
		"POST",
		"/history",
		map[string]string{"refstr": "peer_a/test_movies"},
		remoteLogHandler)
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	actualBody = string(fixTs.ReplaceAll([]byte(actualBody), []byte(`"commitTime":"timeStampHere"`)))
	expectBody := `{"data":[{"username":"peer_a","name":"test_movies","path":"/ipfs/QmXmnH1tFKyG493wsFiisZ14N4cjrymZZ6pqK3Vr9vWS2p","bodySize":720,"commitTime":"timeStampHere","commitTitle":"body changed by 70%","commitMessage":"body:\n\tchanged by 70%"},{"username":"peer_a","name":"test_movies","path":"/ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF","bodySize":224,"commitTime":"timeStampHere","commitTitle":"created dataset from body_ten.csv","commitMessage":"created dataset from body_ten.csv"}],"meta":{"code":200},"pagination":{"page":1,"pageSize":100,"nextUrl":"/history/peer_a/test_movies?page=2\u0026pageSize=100","prevUrl":""}}`
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("body mismatch (-want +got):%s\n", diff)
	}

	// Validates output of fetching from a remote for a remote dataset
	actualStatusCode, actualBody = APICall(
		"POST",
		"/history",
		map[string]string{
			"refstr": "peer_a/test_movies",
			"remote": "a_node",
		},
		localLogHandler)
	if actualStatusCode != 200 {
		t.Errorf("expected status code 200, got %d", actualStatusCode)
	}
	actualBody = string(fixTs.ReplaceAll([]byte(actualBody), []byte(`"commitTime":"timeStampHere"`)))
	expectBody = `{"data":[{"username":"peer_a","name":"test_movies","path":"/ipfs/QmXmnH1tFKyG493wsFiisZ14N4cjrymZZ6pqK3Vr9vWS2p","foreign":true,"bodySize":720,"commitTime":"timeStampHere","commitTitle":"body changed by 70%"},{"username":"peer_a","name":"test_movies","path":"/ipfs/QmNX9ZKXtdskpYSQ5spd1qvqB2CPoWfJbdAcWoFndintrF","foreign":true,"bodySize":224,"commitTime":"timeStampHere","commitTitle":"created dataset from body_ten.csv"}],"meta":{"code":200},"pagination":{"page":1,"pageSize":100,"nextUrl":"/history/peer_a/test_movies?page=2\u0026pageSize=100\u0026remote=a_node","prevUrl":""}}`
	if diff := cmp.Diff(expectBody, actualBody); diff != "" {
		t.Errorf("body mismatch (-want +got):%s\n", diff)
	}
}

// APICall calls the api and returns the status code and body
func APICall(method, reqURL string, params map[string]string, hf http.HandlerFunc) (int, string) {
	req := httptest.NewRequest(method, reqURL, nil)
	ctx, cncl := context.WithTimeout(req.Context(), time.Second*10)
	req.WithContext(ctx)
	defer cncl()

	// add parameters from map
	if params != nil {
		q := req.URL.Query()
		for key := range params {
			q.Add(key, params[key])
		}
		req.URL.RawQuery = q.Encode()
	}

	// Set form-encoded header so server will find the parameters
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
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
