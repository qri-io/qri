package dsutil

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/dataset"
)

func TestFormFileDataset(t *testing.T) {
	r := newFormFileRequest(t, nil, nil)
	dsp := &dataset.Dataset{}
	if err := FormFileDataset(r, dsp); err != nil {
		t.Error("expected 'empty' request to be ok")
	}

	r = newFormFileRequest(t, map[string]string{
		"file":      testdataFile("dstest/testdata/complete/input.dataset.json"),
		"viz":       testdataFile("dstest/testdata/complete/template.html"),
		"transform": testdataFile("dstest/testdata/complete/transform.star"),
		"body":      testdataFile("dstest/testdata/complete/body.csv"),
	}, nil)
	if err := FormFileDataset(r, dsp); err != nil {
		t.Error(err)
	}

	r = newFormFileRequest(t, map[string]string{
		"file": "testdata/dataset.yml",
		"body": testdataFile("dstest/testdata/complete/body.csv"),
	}, nil)
	if err := FormFileDataset(r, dsp); err != nil {
		t.Error(err)
	}
}

func newFormFileRequest(t *testing.T, files, params map[string]string) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for name, path := range files {
		data, err := os.Open(path)
		if err != nil {
			t.Fatalf("error opening datafile: %s %s", name, err)
		}
		dataPart, err := writer.CreateFormFile(name, filepath.Base(path))
		if err != nil {
			t.Fatalf("error adding data file to form: %s %s", name, err)
		}

		if _, err := io.Copy(dataPart, data); err != nil {
			t.Fatalf("error copying data: %s", err)
		}
	}

	for key, val := range params {
		if err := writer.WriteField(key, val); err != nil {
			t.Fatalf("error adding field to writer: %s", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("error closing writer: %s", err)
	}

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	return req
}
