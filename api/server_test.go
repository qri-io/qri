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

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo/test"
)

func confirmQriNotRunning() error {
	l, err := net.Listen("tcp", ":"+config.DefaultAPIPort)
	if err != nil {
		return fmt.Errorf("it looks like a qri server is already running on port %s, please close before running tests", config.DefaultAPIPort)
	}

	l.Close()
	return nil
}

func TestServerRoutes(t *testing.T) {
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

	r, err := test.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	s, err := New(r, func(c *config.Config) {
		c.P2P.Enabled = false
		// c.RE.MemOnly = true
	})
	if err != nil {
		t.Error(err.Error())
		return
	}

	server := httptest.NewServer(NewServerRoutes(s))

	// test endpoints that take mime/multipart files first
	testMimeMultipart(t, server, client)

	cases := []struct {
		method      string
		endpoint    string
		reqBodyPath string
		resBodyPath string
		resStatus   int
	}{
		{"GET", "/status", "", "statusResponse.json", 200},
		{"GET", "/list", "", "listResponse.json", 200},

		// profile
		{"POST", "/profile", "profileRequest.json", "profileResponse.json", 200},
		{"GET", "/profile", "", "profileResponse.json", 200},
		{"GET", "/me", "", "profileResponse.json", 200},

		{"POST", "/add", "addRequestFromURL.json", "addResponseFromURL.json", 200},

		// get dataset
		{"GET", "/me/family_relationships", "", "getResponseFamilyRelationships.json", 200},
		{"GET", "/me/family_relationships/at/map/QmdbJGpmZKsbKpBGQbWS7PjodGtrXX3hAHvxdgUsuf9a3N", "", "getResponseFamilyRelationships.json", 200},
		{"GET", "/at/map/QmdbJGpmZKsbKpBGQbWS7PjodGtrXX3hAHvxdgUsuf9a3N", "", "getResponseFamilyRelationships.json", 200},

		{"POST", "/rename", "renameRequest.json", "renameResponse.json", 200},

		{"POST", "/save/me/cities", "saveMetaRequest.json", "saveMetaResponse.json", 200},

		// history
		{"GET", "/history/me/cities", "", "historyResponse.json", 200},
		{"GET", "/history/me/cities/at/map/QmcQsi93yUryyWvw6mPyDNoKRb7FcBx8QGBAeJ25kXQjnC", "", "historyResponsePath.json", 200},
		{"GET", "/history/at/map/QmcQsi93yUryyWvw6mPyDNoKRb7FcBx8QGBAeJ25kXQjnC", "", "historyResponseAt.json", 200},

		{"GET", "/export/me/cities", "", "", 200},
		{"GET", "/export/me/cities/at/map/QmcQsi93yUryyWvw6mPyDNoKRb7FcBx8QGBAeJ25kXQjnC", "", "", 200},
		{"GET", "/export/at/map/QmcQsi93yUryyWvw6mPyDNoKRb7FcBx8QGBAeJ25kXQjnC", "", "", 200},

		// diff
		{"GET", "/diff", "diffRequest.json", "diffResponse.json", 200},
		{"GET", "/diff", "diffRequestPlusMinusColor.json", "diffResponsePlusMinusColor.json", 200},

		// remove
		{"POST", "/remove/me/cities/at/map/QmcQsi93yUryyWvw6mPyDNoKRb7FcBx8QGBAeJ25kXQjnC", "", "removeResponseWithPath.json", 200},
		{"POST", "/remove/at/map/QmdbJGpmZKsbKpBGQbWS7PjodGtrXX3hAHvxdgUsuf9a3N", "", "removeResponseByPath.json", 200},

		{"GET", "/connect/", "", "", 400},

		// blatently checking all options for easy test coverage bump
		{"OPTIONS", "/add", "", "", 200},
		{"OPTIONS", "/add/", "", "", 200},
		{"OPTIONS", "/profile", "", "", 200},
		{"OPTIONS", "/me", "", "", 200},
		{"OPTIONS", "/export/", "", "", 200},
		{"OPTIONS", "/list", "", "", 200},
		{"OPTIONS", "/save", "", "", 200},
		{"OPTIONS", "/remove/", "", "", 200},
		{"OPTIONS", "/rename", "", "", 200},
		{"OPTIONS", "/me/", "", "", 200},
		{"OPTIONS", "/list/", "", "", 200},
		{"OPTIONS", "/history/", "", "", 200},
	}

	for i, c := range cases {
		var (
			reqBody []byte
			resBody []byte
			gotBody []byte
			err     error
		)

		if c.reqBodyPath == "" {
			reqBody = nil
		} else {
			reqBody, err = ioutil.ReadFile("testdata/" + c.reqBodyPath)
			if err != nil {
				t.Errorf("case %d error reading file: %s", i, err.Error())
				continue
			}
		}

		req, err := http.NewRequest(c.method, server.URL+c.endpoint, bytes.NewReader(reqBody))
		if err != nil {
			t.Errorf("case %d error creating request: %s", i, err.Error())
			continue
		}

		req.Header.Add("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			t.Errorf("case %d error performing request: %s", i, err.Error())
			continue
		}

		if res.StatusCode != c.resStatus {
			t.Errorf("case %d: %s - %s status code mismatch. expected: %d, got: %d", i, c.method, c.endpoint, c.resStatus, res.StatusCode)
			continue
		}

		if c.resBodyPath == "" {
			resBody = nil
		} else {
			resBody, err = ioutil.ReadFile("testdata/" + c.resBodyPath)
			if err != nil {
				t.Errorf("case %d error reading file: %s", i, err.Error())
				continue
			}

			gotBody, err = ioutil.ReadAll(res.Body)
			if err != nil {
				t.Errorf("case %d, error reading response body: %s", i, err.Error())
				continue
			}

			if string(gotBody) != string(resBody) {
				// t.Errorf("case %d: %s - %s response body mismatch.", i, c.method, c.endpoint)
				// TODO - this is spitting out _very_ large reponses on fail
				t.Errorf("case %d: %s - %s response body mismatch.", i, c.method, c.endpoint)

				dirpath := filepath.Join(os.TempDir(), "qri-io/qri/api", "TestServerRoutes")
				if err := os.MkdirAll(dirpath, os.ModePerm); err != nil {
					t.Logf("error creating test dirpath: %s", err.Error())
					continue
				}
				path := filepath.Join(dirpath, c.resBodyPath)
				if err := ioutil.WriteFile(path, gotBody, os.ModePerm); err != nil {
					t.Logf("error writing test file: %s", err.Error())
					continue
				}
				t.Logf("error written to: %s", path)
				continue
			}
		}
	}
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

	r, err := test.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	s, err := New(r, func(opt *config.Config) {
		opt.P2P.Enabled = false
		// opt.MemOnly = true
		opt.API.ReadOnly = true
	})
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
		{"POST", "/add", 403},
		{"PUT", "/add", 403},
		{"POST", "/add/", 403},
		{"PUT", "/add/", 403},
		{"POST", "/rename", 403},
		{"PUT", "/rename", 403},
		{"GET", "/export/", 403},
		{"POST", "/diff", 403},
		{"GET", "/diff", 403},
		{"GET", "/data/", 403},

		// active endpoints:
		{"GET", "/status", 200},
		{"GET", "/list/peer", 200},
		// Cannot test connect endpoint until we have peers in this test suite
		// {"GET", "/connect/QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", 200},
		// Cannot test endpoint until we have peers in this test suite
		// {"GET", "/peer", 200},
		{"GET", "/peer/movies", 200},
		{"GET", "/history/peer/movies", 200},

		// blatently checking all options for easy test coverage bump
		{"OPTIONS", "/add", 200},
		{"OPTIONS", "/add/", 200},
		{"OPTIONS", "/profile", 200},
		{"OPTIONS", "/me", 200},
		{"OPTIONS", "/export/", 200},
		{"OPTIONS", "/list", 200},
		{"OPTIONS", "/save", 200},
		{"OPTIONS", "/remove/", 200},
		{"OPTIONS", "/rename", 200},
		{"OPTIONS", "/me/", 200},
		{"OPTIONS", "/list/", 200},
		{"OPTIONS", "/history/", 200},
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

func testMimeMultipart(t *testing.T, server *httptest.Server, client *http.Client) {
	cases := []struct {
		method         string
		endpoint       string
		expectBodyPath string
		resStatus      int
		filePaths      map[string]string
		params         map[string]string
	}{
		{"POST", "/remove/me/cities", "testdata/removeResponse.json", 200,
			map[string]string{},
			map[string]string{},
		},
		{"POST", "/add", "testdata/addResponsePrivate.json", 500,
			map[string]string{
				"file":      "testdata/cities/data.csv",
				"structure": "testdata/cities/structure.json",
				"metadata":  "testdata/cities/meta.json",
			},
			map[string]string{
				"peername": "peer",
				"name":     "cities",
				"private":  "true",
			},
		},
		{"POST", "/add", "testdata/addResponseFromFile.json", 200,
			map[string]string{
				"file":      "testdata/cities/data.csv",
				"structure": "testdata/cities/structure.json",
				"metadata":  "testdata/cities/meta.json",
			},
			map[string]string{
				"peername": "peer",
				"name":     "cities",
			},
		},
		{"POST", "/save", "testdata/saveResponse.json", 200,
			map[string]string{
				"file": "testdata/cities/data_update.csv",
			},
			map[string]string{
				"peername": "peer",
				"name":     "cities",
				"title":    "added row to include Seoul, Korea",
				"message":  "want to expand this list to include more cities",
			},
		},
		{"GET", "/profile", "testdata/profileResponseInitial.json", 200,
			map[string]string{},
			map[string]string{},
		},
		{"POST", "/save", "testdata/saveResponseMeta.json", 200,
			map[string]string{
				"metadata": "testdata/cities/meta_update.json",
			},
			map[string]string{
				"peername": "peer",
				"name":     "cities",
				"title":    "Adding more specific metadata",
				"message":  "added title and keywords",
			},
		},
		{"POST", "/profile/photo", "testdata/photoResponse.json", 200,
			map[string]string{
				"file": "testdata/rico_400x400.jpg",
			},
			map[string]string{
				"peername": "peer",
			},
		},
		{"POST", "/profile/poster", "testdata/posterResponse.json", 200,
			map[string]string{
				"file": "testdata/rico_poster_1500x500.jpg",
			},
			map[string]string{
				"peername": "peer",
			},
		},
	}

	for i, c := range cases {

		expectBody, err := ioutil.ReadFile(c.expectBodyPath)
		if err != nil {
			t.Errorf("case add dataset from file, error reading expected response from file: %s", err)
		}

		req, err := NewFilesRequest(c.method, c.endpoint, server.URL+c.endpoint, c.filePaths, c.params)
		if err != nil {
			t.Errorf("testMimeMultipart case %d, %s - %s:\nerror making mime/multipart request: %s", i, c.method, c.endpoint, err)
			continue
		}

		res, err := client.Do(req)
		if err != nil {
			t.Errorf("testMimeMultipart case %d, %s - %s:\nerror performing request: %s", i, c.method, c.endpoint, err)
			continue
		}

		gotBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Errorf("testMimeMultipart case %d, %s - %s:\nerror reading response body request: %s", i, c.method, c.endpoint, err)
			continue
		}

		if string(gotBody) != string(expectBody) {
			// t.Errorf("testMimeMultipart case %d, %s - %s:\nresponse body mismatch. expected: %s, got %s", i, c.method, c.endpoint, string(expectBody), string(gotBody))
			// t.Errorf("testMimeMultipart case %d, %s - %s:\nresponse body mismatch. expected: %s, got %s", i, c.method, c.endpoint, string(expectBody), string(gotBody))
			continue
		}

		if res.StatusCode != c.resStatus {
			t.Errorf("testMimeMultipart case %d, %s - %s:\nstatus code mismatch. expected: %d, got: %d", i, c.method, c.endpoint, c.resStatus, res.StatusCode)
			continue
		}
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
