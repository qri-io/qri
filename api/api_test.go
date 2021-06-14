package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/beme/abide"
	"github.com/gorilla/mux"
	golog "github.com/ipfs/go-log"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	apispec "github.com/qri-io/qri/api/spec"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/test"
)

func init() {
	abide.SnapshotsDir = "testdata"
}

func TestMain(m *testing.M) {
	exit := m.Run()
	abide.Cleanup()
	os.Exit(exit)
}

func TestApiSpec(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Fatal(err.Error())
	}

	tr := NewAPITestRunner(t)
	defer tr.Delete()

	ts := tr.MustTestServer(t)
	defer ts.Close()

	apispec.AssertHTTPAPISpec(t, ts.URL, "./spec")
}

func TestConnectNoP2P(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node, teardown := newTestNode(t)
	defer teardown()

	inst := newTestInstanceWithProfileFromNode(ctx, node)
	cfg := inst.GetConfig()
	cfg.P2P.Enabled = false
	if err := inst.ChangeConfig(cfg); err != nil {
		t.Fatal(err)
	}

	s := New(inst)
	ctx, cancel2 := context.WithTimeout(ctx, time.Millisecond*15)
	defer cancel2()

	if err := s.Serve(ctx); !errors.Is(err, http.ErrServerClosed) {
		t.Fatal(err)
	}
}

func newTestRepo(t *testing.T) (r repo.Repo, teardown func()) {
	var err error
	if err = confirmQriNotRunning(); err != nil {
		t.Fatal(err.Error())
	}

	// bump up log level to keep test output clean
	golog.SetLogLevel("qriapi", "error")

	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	logbookTsSec := 0
	prevLogbookTs := logbook.NewTimestamp
	logbook.NewTimestamp = func() int64 {
		logbookTsSec++
		return time.Date(2001, 01, 01, 01, 01, logbookTsSec, 01, time.UTC).Unix()
	}

	if r, err = test.NewTestRepo(); err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}

	teardown = func() {
		golog.SetLogLevel("qriapi", "info")
		// lib.SaveConfig = prevSaveConfig
		dsfs.Timestamp = prevTs
		logbook.NewTimestamp = prevLogbookTs
	}

	return
}

func newTestNode(t *testing.T) (node *p2p.QriNode, teardown func()) {
	t.Helper()

	var r repo.Repo
	r, teardown = newTestRepo(t)
	node, err := p2p.NewQriNode(r, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	return node, teardown
}

func testConfigAndSetter() (cfg *config.Config, setCfg func(*config.Config) error) {
	cfg = testcfg.DefaultConfigForTesting()
	cfg.Profile = test.ProfileConfig()

	setCfg = func(*config.Config) error { return nil }
	return
}

func newTestInstanceWithProfileFromNode(ctx context.Context, node *p2p.QriNode) *lib.Instance {
	cfg := testcfg.DefaultConfigForTesting()
	cfg.Profile, _ = node.Repo.Profiles().Owner().Encode()
	return lib.NewInstanceFromConfigAndNode(ctx, cfg, node)
}

type handlerTestCase struct {
	method, endpoint string
	body             []byte
	muxVars          map[string]string
}

// runHandlerTestCases executes a slice of handlerTestCase against a handler
func runHandlerTestCases(t *testing.T, name string, h http.HandlerFunc, cases []handlerTestCase, jsonHeader bool) {
	for i, c := range cases {
		name := fmt.Sprintf("%s %s case %d: %s %s", t.Name(), name, i, c.method, c.endpoint)
		req := httptest.NewRequest(c.method, c.endpoint, bytes.NewBuffer(c.body))
		if c.muxVars != nil {
			req = mux.SetURLVars(req, c.muxVars)
		}
		setRefStringFromMuxVars(req)
		if jsonHeader {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()

		h(w, req)

		res := w.Result()
		abide.AssertHTTPResponse(t, name, res)
	}
}

// runHandlerZipPostTestCases executes a slice of handlerTestCase against a handler using zip content-type
func runHandlerZipPostTestCases(t *testing.T, name string, h http.HandlerFunc, cases []handlerTestCase) {
	for i, c := range cases {
		name := fmt.Sprintf("%s %s case %d: %s %s", t.Name(), name, i, c.method, c.endpoint)
		req := httptest.NewRequest(c.method, c.endpoint, bytes.NewBuffer(c.body))
		req.Header.Set("Content-Type", "application/zip")
		w := httptest.NewRecorder()

		h(w, req)

		res := w.Result()
		abide.AssertHTTPResponse(t, name, res)
	}
}

// mustFile reads file bytes, calling t.Fatalf if the file doesn't exist
func mustFile(t *testing.T, filename string) []byte {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("error opening test file: %s: %s", filename, err.Error())
	}
	return data
}

func confirmQriNotRunning() error {
	addr, err := ma.NewMultiaddr(config.DefaultAPIAddress)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	l, err := manet.Listen(addr)
	if err != nil {
		return fmt.Errorf("it looks like a qri server is already running on address %s, please close before running tests", config.DefaultAPIAddress)
	}

	l.Close()
	return nil
}

func TestHealthCheck(t *testing.T) {
	prevAPIVer := APIVersion
	APIVersion = "test_version"
	defer func() {
		APIVersion = prevAPIVer
	}()

	healthCheckCases := []handlerTestCase{
		{"GET", "/", nil, nil},
	}
	runHandlerTestCases(t, "health check", HealthCheckHandler, healthCheckCases, true)
}

type handlerMimeMultipartTestCase struct {
	method    string
	endpoint  string
	filePaths map[string]string
	params    map[string]string
	muxVars   map[string]string
}

func runMimeMultipartHandlerTestCases(t *testing.T, name string, h http.HandlerFunc, cases []handlerMimeMultipartTestCase) {
	for i, c := range cases {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		name := fmt.Sprintf("%s %s case %d: %s %s", t.Name(), name, i, c.method, c.endpoint)

		for name, path := range c.filePaths {
			data, err := os.Open(path)
			if err != nil {
				t.Fatalf("error opening datafile: %s %s", name, err)
			}
			dataPart, err := writer.CreateFormFile(name, filepath.Base(path))
			if err != nil {
				t.Fatalf("error adding data file to form: %s %s", name, err)
			}

			if _, err := io.Copy(dataPart, data); err != nil {
				t.Fatalf("error copying data: %s %s %s", c.method, c.endpoint, err)
			}
		}
		for key, val := range c.params {
			if err := writer.WriteField(key, val); err != nil {
				t.Fatalf("error adding field to writer: %s %s", name, err)
			}
		}

		if err := writer.Close(); err != nil {
			t.Fatalf("error closing writer: %s", err)
		}

		req := httptest.NewRequest(c.method, c.endpoint, body)
		req.Header.Add("Content-Type", writer.FormDataContentType())
		if c.muxVars != nil {
			req = mux.SetURLVars(req, c.muxVars)
		}
		setRefStringFromMuxVars(req)

		w := httptest.NewRecorder()

		h(w, req)

		res := w.Result()
		abide.AssertHTTPResponse(t, name, res)
	}
}

// NewFilesRequest creates a mime/multipart http.Request with files specified by a map of param : filepath,
// and form values specified by a map, params
func NewFilesRequest(method, endpoint, url string, filePaths, params map[string]string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for name, path := range filePaths {
		data, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("error opening datafile: %s %s %s", method, endpoint, err)
		}
		dataPart, err := writer.CreateFormFile(name, filepath.Base(path))
		if err != nil {
			return nil, fmt.Errorf("error adding data file to form: %s %s %s", method, endpoint, err)
		}

		if _, err := io.Copy(dataPart, data); err != nil {
			return nil, fmt.Errorf("error copying data: %s %s %s", method, endpoint, err)
		}
	}
	for key, val := range params {
		if err := writer.WriteField(key, val); err != nil {
			return nil, fmt.Errorf("error adding field to writer: %s %s %s", method, endpoint, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("error closing writer: %s", err)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %s %s %s", method, endpoint, err)
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())

	return req, nil
}
