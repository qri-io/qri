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

	"github.com/beme/abide"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
)

func TestProfileHandler(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

	cases := []handlerTestCase{
		{"OPTIONS", "/", nil},
		{"GET", "/", nil},
		{"POST", "/", mustFile(t, "testdata/profileRequest.json")},
		{"POST", "/", []byte(``)},
		{"DELETE", "/", nil},
	}

	inst := newTestInstanceWithProfileFromNode(node)
	proh := NewProfileHandlers(inst, false)
	runHandlerTestCases(t, "profile", proh.ProfileHandler, cases, true)

	readOnlyCases := []handlerTestCase{
		{"GET", "/", nil},
	}
	proh.ReadOnly = true
	runHandlerTestCases(t, "read-only", proh.ProfileHandler, readOnlyCases, true)

	mimeCases := []handlerMimeMultipartTestCase{
		{"GET", "/",
			map[string]string{},
			map[string]string{},
		},
	}
	runMimeMultipartHandlerTestCases(t, "mime read", proh.ProfileHandler, mimeCases)
}

func TestProfilePhotoHandler(t *testing.T) {
	node, teardown := newTestNode(t)
	defer teardown()

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

	cfg := config.DefaultConfigForTesting()
	// newTestNode uses a different profile. assign here so instance config.Profile
	// node config.Profile match
	pro, _ := node.Repo.Profile()
	cfg.Profile, _ = pro.Encode()

	// TODO (b5) - hack until tests have better instance-generation primitives
	inst := lib.NewInstanceFromConfigAndNode(cfg, node)
	proh := NewProfileHandlers(inst, false)

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
	node, teardown := newTestNode(t)
	defer teardown()

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

	cfg := config.DefaultConfigForTesting()
	// newTestNode uses a different profile. assign here so instance config.Profile
	// node config.Profile match
	pro, _ := node.Repo.Profile()
	cfg.Profile, _ = pro.Encode()

	// TODO (b5) - hack until tests have better instance-generation primitives
	inst := lib.NewInstanceFromConfigAndNode(cfg, node)
	proh := NewProfileHandlers(inst, false)

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
