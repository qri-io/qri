package repo

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/dataset"
)

func TestDatasetPodBodyFile(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"json":"data"}`))
	}))
	badS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	cases := []struct {
		dsp      *dataset.DatasetPod
		filename string
		fileLen  int
		err      string
	}{
		// bad input
		{&dataset.DatasetPod{}, "", 0, "not found"},

		// inline data
		{&dataset.DatasetPod{BodyBytes: []byte("a,b,c\n1,2,3")}, "", 0, "specifying bodyBytes requires format be specified in dataset.structure"},
		{&dataset.DatasetPod{Structure: &dataset.StructurePod{Format: "csv"}, BodyBytes: []byte("a,b,c\n1,2,3")}, "body.csv", 11, ""},

		// urlz
		{&dataset.DatasetPod{BodyPath: "http://"}, "", 0, "fetching body url: Get http:: http: no Host in request URL"},
		{&dataset.DatasetPod{BodyPath: fmt.Sprintf("%s/foobar.json", badS.URL)}, "", 0, "invalid status code fetching body url: 500"},
		{&dataset.DatasetPod{BodyPath: fmt.Sprintf("%s/foobar.json", s.URL)}, "foobar.json", 15, ""},

		// local filepaths
		{&dataset.DatasetPod{BodyPath: "nope.cbor"}, "", 0, "reading body file: open nope.cbor: no such file or directory"},
		{&dataset.DatasetPod{BodyPath: "nope.yaml"}, "", 0, "reading body file: open nope.yaml: no such file or directory"},
		{&dataset.DatasetPod{BodyPath: "testdata/schools.cbor"}, "schools.cbor", 154, ""},
		{&dataset.DatasetPod{BodyPath: "testdata/bad.yaml"}, "", 0, "converting yaml body to json: yaml: line 1: did not find expected '-' indicator"},
		{&dataset.DatasetPod{BodyPath: "testdata/oh_hai.yaml"}, "oh_hai.json", 29, ""},
	}

	for i, c := range cases {
		file, err := DatasetPodBodyFile(c.dsp)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if file == nil && c.filename != "" {
			t.Errorf("case %d expected file", i)
			continue
		} else if c.filename == "" {
			continue
		}

		if c.filename != file.FileName() {
			t.Errorf("case %d filename mismatch. expected: '%s', got: '%s'", i, c.filename, file.FileName())
		}
		data, err := ioutil.ReadAll(file)
		if err != nil {
			t.Errorf("case %d error reading file: %s", i, err.Error())
			continue
		}
		if c.fileLen != len(data) {
			t.Errorf("case %d file length mismatch. expected: %d, got: %d", i, c.fileLen, len(data))
		}

		if err := file.Close(); err != nil {
			t.Errorf("case %d error closing file: %s", i, err.Error())
		}
	}
}
