package api

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
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
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestProfileHandler(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// bump up log level to keep test output clean
	golog.SetLogLevel("qriapi", "error")
	defer golog.SetLogLevel("qriapi", "info")

	// use a test registry server & client & client
	rc, registryServer := regmock.NewMockServer()
	// we need to artificially specify the timestamp
	// we use the dsfs.Timestamp func variable to override
	// the actual time
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r, err := test.NewTestRepo(rc)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	core.Config = config.DefaultConfig()
	core.Config.Profile = test.ProfileConfig()
	core.Config.Registry.Location = registryServer.URL
	prevSaveConfig := core.SaveConfig
	core.SaveConfig = func() error {
		p, err := profile.NewProfile(core.Config.Profile)
		if err != nil {
			return err
		}

		r.SetProfile(p)
		return err
	}
	defer func() { core.SaveConfig = prevSaveConfig }()

	if err != nil {
		t.Error(err.Error())
		return
	}

	cases := []struct {
		name, method, endpoint string
		body                   []byte
		readOnly               bool
	}{
		{"OPTIONS", "OPTIONS", "/", nil, false},
		{"GET", "GET", "/", nil, false},
		{"GET read-only", "GET", "/", nil, true},
		{"POST", "POST", "/", []byte(`{"id": "","created": "0001-01-01T00:00:00Z","updated": "0001-01-01T00:00:00Z","peername": "","type": "peer","email": "test@email.com","name": "test name","description": "test description","homeUrl": "http://test.url","color": "default","thumb": "/","profile": "/","poster": "/","twitter": "test"}`), false},
		{"POST bad data", "POST", "/", []byte(``), false},
		{"bad method", "DELETE", "/", nil, false},
	}

	proh := NewProfileHandlers(r, false)

	for _, c := range cases {
		name := fmt.Sprintf("Profile Test: %s", c.name)
		req := httptest.NewRequest(c.method, c.endpoint, bytes.NewBuffer(c.body))

		w := httptest.NewRecorder()

		proh.ReadOnly = c.readOnly
		proh.ProfileHandler(w, req)

		res := w.Result()
		abide.AssertHTTPResponse(t, name, res)
	}
}

func TestProfilePhotoHandler(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// bump up log level to keep test output clean
	golog.SetLogLevel("qriapi", "error")
	defer golog.SetLogLevel("qriapi", "info")

	// use a test registry server & client
	rc, registryServer := regmock.NewMockServer()

	// in order to have consistent responses
	// we need to artificially specify the timestamp
	// we use the dsfs.Timestamp func variable to override
	// the actual time
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r, err := test.NewTestRepo(rc)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	core.Config = config.DefaultConfig()
	core.Config.Profile = test.ProfileConfig()
	core.Config.Registry.Location = registryServer.URL
	prevSaveConfig := core.SaveConfig
	core.SaveConfig = func() error {
		p, err := profile.NewProfile(core.Config.Profile)
		if err != nil {
			return err
		}

		r.SetProfile(p)
		return err
	}
	defer func() { core.SaveConfig = prevSaveConfig }()

	if err != nil {
		t.Error(err.Error())
		return
	}

	cases := []struct {
		name, method, endpoint string
		filepaths              map[string]string
		params                 map[string]string
	}{
		{"OPTIONS", "OPTIONS", "/", nil, nil},
		{"POST", "POST", "/",
			map[string]string{
				"file": "testdata/rico_400x400.jpg",
			},
			nil,
		},
		{"GET", "GET", "/", nil,
			map[string]string{
				"peername": "peer",
			},
		},
		{"POST bad file format", "POST", "/",
			map[string]string{
				"file": "testdata/cities/data.csv",
			},
			nil,
		},
	}

	proh := NewProfileHandlers(r, false)

	for _, c := range cases {
		name := fmt.Sprintf("Profile Photo Test: %s", c.name)

		req, err := NewFilesTestRequest(c.method, c.endpoint, c.endpoint, c.filepaths, c.params)
		if err != nil {
			t.Errorf("testMimeMultipart: %s:\nerror making mime/multipart request: %s", c.name, err)
			continue
		}

		w := httptest.NewRecorder()

		proh.ProfilePhotoHandler(w, req)

		res := w.Result()
		abide.AssertHTTPResponse(t, name, res)
	}
}

func TestProfilePosterHandler(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	// bump up log level to keep test output clean
	golog.SetLogLevel("qriapi", "error")
	defer golog.SetLogLevel("qriapi", "info")

	// use a test registry server & client
	rc, registryServer := regmock.NewMockServer()

	// in order to have consistent responses
	// we need to artificially specify the timestamp
	// we use the dsfs.Timestamp func variable to override
	// the actual time
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	r, err := test.NewTestRepo(rc)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	core.Config = config.DefaultConfig()
	core.Config.Profile = test.ProfileConfig()
	core.Config.Registry.Location = registryServer.URL
	prevSaveConfig := core.SaveConfig
	core.SaveConfig = func() error {
		p, err := profile.NewProfile(core.Config.Profile)
		if err != nil {
			return err
		}

		r.SetProfile(p)
		return err
	}
	defer func() { core.SaveConfig = prevSaveConfig }()

	if err != nil {
		t.Error(err.Error())
		return
	}

	cases := []struct {
		name, method, endpoint string
		filepaths              map[string]string
		params                 map[string]string
	}{
		{"OPTIONS", "OPTIONS", "/", nil, nil},
		{"POST", "POST", "/",
			map[string]string{
				"file": "testdata/rico_poster_1500x500.jpg",
			},
			nil,
		},
		{"GET", "GET", "/", nil,
			map[string]string{
				"peername": "peer",
				"id":       "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
			},
		},
		{"POST bad file format", "POST", "/",
			map[string]string{
				"file": "testdata/cities/data.csv",
			},
			nil,
		},
	}

	proh := NewProfileHandlers(r, false)

	for _, c := range cases {
		name := fmt.Sprintf("Profile Poster Test: %s", c.name)

		req, err := NewFilesTestRequest(c.method, c.endpoint, c.endpoint, c.filepaths, c.params)
		if err != nil {
			t.Errorf("testMimeMultipart: %s:\nerror making mime/multipart request: %s", c.name, err)
			continue
		}
		w := httptest.NewRecorder()

		proh.PosterHandler(w, req)

		res := w.Result()
		abide.AssertHTTPResponse(t, name, res)
	}
}

// NewFilesTestRequest creates a mime/multipart httptest.Request with files specified by a map of param : filepath,
// and form values specified by a map, params
func NewFilesTestRequest(method, endpoint, url string, filepaths, params map[string]string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for name, path := range filepaths {
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

	req := httptest.NewRequest(method, url, body)

	req.Header.Add("Content-Type", writer.FormDataContentType())

	return req, nil
}
