package api

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/beme/abide"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func init() {
	abide.SnapshotsDir = "testdata"
}

func newTestRepo(t *testing.T) (r repo.Repo, teardown func()) {
	var err error
	if err = confirmQriNotRunning(); err != nil {
		t.Fatalf(err.Error())
	}

	// bump up log level to keep test output clean
	golog.SetLogLevel("qriapi", "error")

	// use a test registry server & client & client
	rc, registryServer := regmock.NewMockServer()
	// we need to artificially specify the timestamp
	// we use the dsfs.Timestamp func variable to override
	// the actual time
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	if r, err = test.NewTestRepo(rc); err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}

	lib.Config = config.DefaultConfigForTesting()
	lib.Config.Profile = test.ProfileConfig()
	lib.Config.Registry.Location = registryServer.URL
	prevSaveConfig := lib.SaveConfig
	lib.SaveConfig = func() error {
		p, err := profile.NewProfile(lib.Config.Profile)
		if err != nil {
			return err
		}

		r.SetProfile(p)
		return err
	}

	teardown = func() {
		golog.SetLogLevel("qriapi", "info")
		lib.SaveConfig = prevSaveConfig
		dsfs.Timestamp = prevTs
	}

	return
}

type handlerTestCase struct {
	method, endpoint string
	body             []byte
}

// runHandlerTestCases executes a slice of handlerTestCase against a handler
func runHandlerTestCases(t *testing.T, name string, h http.HandlerFunc, cases []handlerTestCase) {
	for i, c := range cases {
		name := fmt.Sprintf("%s %s case %d: %s %s", t.Name(), name, i, c.method, c.endpoint)
		req := httptest.NewRequest(c.method, c.endpoint, bytes.NewBuffer(c.body))
		// TODO - make this settable with some sort of test case interface
		req.Header.Set("Content-Type", "application/json")
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
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", config.DefaultAPIPort))
	if err != nil {
		return fmt.Errorf("it looks like a qri server is already running on port %d, please close before running tests", config.DefaultAPIPort)
	}

	l.Close()
	return nil
}

func TestServerRoutes(t *testing.T) {
	r, teardown := newTestRepo(t)
	defer teardown()

	h := NewRootHandler(NewDatasetHandlers(r, false), NewPeerHandlers(r, nil, false))
	rootCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
	}
	runHandlerTestCases(t, "root", h.Handler, rootCases)

	healthCheckCases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
	}
	runHandlerTestCases(t, "health check", HealthCheckHandler, healthCheckCases)
}

func TestServerReadOnlyRoutes(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// bump up log level to keep test output clean
	golog.SetLogLevel("qriapi", "error")
	defer golog.SetLogLevel("qriapi", "info")

	// in order to have consistent responses
	// we need to artificially specify the timestamp
	// we use the dsfs.Timestamp func variable to override
	// the actual time
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	client := &http.Client{}

	r, err := test.NewTestRepo(nil)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	cfg := config.DefaultConfigForTesting()
	cfg.P2P.Enabled = false
	cfg.API.ReadOnly = true
	defer func() {
		cfg.P2P.Enabled = true
		cfg.API.ReadOnly = false
	}()

	s, err := New(r, cfg)
	if err != nil {
		t.Error(err.Error())
		return
	}

	server := httptest.NewServer(NewServerRoutes(s))

	cases := []struct {
		method    string
		endpoint  string
		resStatus int
	}{
		// forbidden endpoints
		{"GET", "/ipfs/", 403},
		{"GET", "/ipns/", 403},
		{"GET", "/profile", 403},
		{"POST", "/profile", 403},
		{"GET", "/me", 403},
		{"POST", "/me", 403},
		{"POST", "/profile/photo", 403},
		{"PUT", "/profile/photo", 403},
		{"POST", "/profile/poster", 403},
		{"PUT", "/profile/poster", 403},
		{"GET", "/peers", 403},
		{"GET", "/peers/", 403},
		{"GET", "/connections", 403},
		{"GET", "/list", 403},
		{"POST", "/save", 403},
		{"PUT", "/save", 403},
		{"POST", "/save/", 403},
		{"PUT", "/save/", 403},
		{"POST", "/remove/", 403},
		{"DELETE", "/remove/", 403},
		{"GET", "/me/", 403},
		{"POST", "/new", 403},
		{"PUT", "/new", 403},
		{"POST", "/add/", 403},
		{"PUT", "/add/", 403},
		{"POST", "/rename", 403},
		{"PUT", "/rename", 403},
		{"GET", "/export/", 403},
		{"POST", "/diff", 403},
		{"GET", "/diff", 403},
		{"GET", "/body/", 403},
		{"POST", "/registry/", 403},

		// active endpoints:
		{"GET", "/status", 200},
		{"GET", "/list/peer", 200},
		// Cannot test connect endpoint until we have peers in this test suite
		// {"GET", "/connect/QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", 200},
		// Cannot test endpoint until we have peers in this test suite
		// {"GET", "/peer", 200},
		{"GET", "/peer/movies", 200},
		{"GET", "/history/peer/movies", 200},
	}

	for i, c := range cases {

		req, err := http.NewRequest(c.method, server.URL+c.endpoint, nil)
		if err != nil {
			t.Errorf("case %d error creating request: %s", i, err.Error())
			continue
		}

		res, err := client.Do(req)
		if err != nil {
			t.Errorf("case %d error performing request: %s", i, err.Error())
			continue
		}

		if res.StatusCode != c.resStatus {
			t.Errorf("case %d: %s - %s status code mismatch. expected: %d, got: %d", i, c.method, c.endpoint, c.resStatus, res.StatusCode)
			continue
		}
	}
}

type handlerMimeMultipartTestCase struct {
	method    string
	endpoint  string
	filePaths map[string]string
	params    map[string]string
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

		w := httptest.NewRecorder()

		h(w, req)

		res := w.Result()
		abide.AssertHTTPResponse(t, name, res)
	}
}

func testMimeMultipart(t *testing.T, server *httptest.Server, client *http.Client) {

	// cases := []struct {
	// 	method    string
	// 	endpoint  string
	// 	resStatus int
	// 	filePaths map[string]string
	// 	params    map[string]string
	// }{
	// 	// {"POST", "/remove/me/cities", 200,
	// 	// 	map[string]string{},
	// 	// 	map[string]string{},
	// 	// },
	// 	// {"POST", "/new", 500,
	// 	// 	map[string]string{
	// 	// 		"body":      "testdata/cities/data.csv",
	// 	// 		"structure": "testdata/cities/structure.json",
	// 	// 		"metadata":  "testdata/cities/meta.json",
	// 	// 	},
	// 	// 	map[string]string{
	// 	// 		"peername": "peer",
	// 	// 		"name":     "cities",
	// 	// 		"private":  "true",
	// 	// 	},
	// 	// },
	// 	// {"POST", "/new", 200,
	// 	// 	map[string]string{
	// 	// 		"body": "testdata/cities/data.csv",
	// 	// 		"file": "testdata/cities/init_dataset.json",
	// 	// 	},
	// 	// 	map[string]string{
	// 	// 		"peername": "peer",
	// 	// 		"name":     "cities",
	// 	// 	},
	// 	// },
	// 	// {"POST", "/save", "testdata/saveResponse.json", 200,
	// 	// 	map[string]string{
	// 	// 		"body": "testdata/cities/data_update.csv",
	// 	// 	},
	// 	// 	map[string]string{
	// 	// 		"peername": "peer",
	// 	// 		"name":     "cities",
	// 	// 		"title":    "added row to include Seoul, Korea",
	// 	// 		"message":  "want to expand this list to include more cities",
	// 	// 	},
	// 	// },
	// 	{"GET", "/profile", 200,
	// 		map[string]string{},
	// 		map[string]string{},
	// 	},
	// 	// {"POST", "/save", "testdata/saveResponseMeta.json", 200,
	// 	// 	map[string]string{
	// 	// 		"metadata": "testdata/cities/meta_update.json",
	// 	// 	},
	// 	// 	map[string]string{
	// 	// 		"peername": "peer",
	// 	// 		"name":     "cities",
	// 	// 		"title":    "Adding more specific metadata",
	// 	// 		"message":  "added title and keywords",
	// 	// 	},
	// 	// },
	// 	{"POST", "/profile/photo", 200,
	// 		map[string]string{
	// 			"file": "testdata/rico_400x400.jpg",
	// 		},
	// 		map[string]string{
	// 			"peername": "peer",
	// 		},
	// 	},
	// 	{"POST", "/profile/poster", 200,
	// 		map[string]string{
	// 			"file": "testdata/rico_poster_1500x500.jpg",
	// 		},
	// 		map[string]string{
	// 			"peername": "peer",
	// 		},
	// 	},
	// }

	// for i, c := range cases {

	// 	req, err := NewFilesRequest(name, c.endpoint, c.filePaths, c.params)
	// 	if err != nil {
	// 		t.Errorf("testMimeMultipart case %d, %s - %s:\nerror making mime/multipart request: %s", i, name, err)
	// 		continue
	// 	}

	// 	res, err := client.Do(req)
	// 	if err != nil {
	// 		t.Errorf("testMimeMultipart case %d, %s - %s:\nerror performing request: %s", i, name, err)
	// 		continue
	// 	}

	// 	gotBody, err := ioutil.ReadAll(res.Body)
	// 	if err != nil {
	// 		t.Errorf("testMimeMultipart case %d, %s - %s:\nerror reading response body request: %s", i, name, err)
	// 		continue
	// 	}

	// 	if res.StatusCode != c.resStatus {
	// 		t.Errorf("testMimeMultipart case %d, %s - %s:\nstatus code mismatch. expected: %d, got: %d", i, name, c.resStatus, res.StatusCode)
	// 		continue
	// 	}

	// 	name := fmt.Sprintf("%s multipart case %d: %s %s", t.Name(), i, name)
	// 	req := httptest.NewRequest(name, bytes.NewBuffer(c.body))
	// 	// TODO - make this settable with some sort of test case interface
	// 	// req.Header.Set("Content-Type", "application/json")
	// 	w := httptest.NewRecorder()

	// 	// h(w, req)

	// 	res := w.Result()
	// 	abide.AssertHTTPResponse(t, name, res)
	// }
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
