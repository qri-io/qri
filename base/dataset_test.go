package base

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestOpenCloseDataset(t *testing.T) {
	t.Skip("TODO (b5)")
}

func TestListDatasets(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	// Limit to one
	res, err := ListDatasets(ctx, r, "", 1, 0, false, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 1 {
		t.Error("expected one dataset response")
	}

	// Limit to published datasets
	res, err = ListDatasets(ctx, r, "", 1, 0, false, true, false)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 0 {
		t.Error("expected no published datasets")
	}

	if err := SetPublishStatus(r, &ref, true); err != nil {
		t.Fatal(err)
	}

	// Limit to published datasets, after publishing cities
	res, err = ListDatasets(ctx, r, "", 1, 0, false, true, false)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 1 {
		t.Error("expected one published dataset response")
	}

	// Limit to datasets with "city" in their name
	res, err = ListDatasets(ctx, r, "city", 1, 0, false, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 0 {
		t.Error("expected no datasets with \"city\" in their name")
	}

	// Limit to datasets with "cit" in their name
	res, err = ListDatasets(ctx, r, "cit", 1, 0, false, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 1 {
		t.Error("expected one dataset with \"cit\" in their name")
	}
}

func TestCreateDataset(t *testing.T) {
	ctx := context.Background()
	streams := ioes.NewDiscardIOStreams()
	store := cafs.NewMapstore()
	r, err := repo.NewMemRepo(testPeerProfile, store, qfs.NewMemFS(), profile.NewMemStore())
	if err != nil {
		t.Fatal(err.Error())
	}

	ds := &dataset.Dataset{
		Name:   "foo",
		Meta:   &dataset.Meta{Title: "test"},
		Commit: &dataset.Commit{Title: "hello"},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	if _, err := CreateDataset(ctx, r, streams, &dataset.Dataset{}, &dataset.Dataset{}, false, true, false, true); err == nil {
		t.Error("expected bad dataset to error")
	}

	ref, err := CreateDataset(ctx, r, streams, ds, &dataset.Dataset{}, false, true, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	refs, err := r.References(0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Errorf("ref length mismatch. expected 1, got: %d", len(refs))
	}

	ds.Meta.Title = "an update"
	ds.PreviousPath = ref.Path
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	prev := ref.Dataset

	ref, err = CreateDataset(ctx, r, streams, ds, prev, false, true, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	refs, err = r.References(0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Errorf("ref length mismatch. expected 1, got: %d", len(refs))
	}

	ds.PreviousPath = ref.Path
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))
	prev = ref.Dataset

	if ref, err = CreateDataset(ctx, r, streams, ds, prev, false, true, false, true); err == nil {
		t.Error("expected unchanged dataset with no force flag to error")
	}

	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))
	if ref, err = CreateDataset(ctx, r, streams, ds, prev, false, true, true, true); err != nil {
		t.Errorf("unexpected force-save error: %s", err)
	}
}

func TestFetchDataset(t *testing.T) {
	ctx := context.Background()
	r1 := newTestRepo(t)
	r2 := newTestRepo(t)
	ref := addCitiesDataset(t, r2)

	// Connect in memory Mapstore's behind the scene to simulate IPFS-like behavior.
	r1.Store().(*cafs.MapStore).AddConnection(r2.Store().(*cafs.MapStore))

	if err := FetchDataset(ctx, r1, &repo.DatasetRef{Peername: "foo", Name: "bar"}, true, true); err == nil {
		t.Error("expected add of invalid ref to error")
	}

	if err := FetchDataset(ctx, r1, &ref, true, true); err != nil {
		t.Error(err.Error())
	}
}

func TestDatasetPodBodyFile(t *testing.T) {
	ctx := context.Background()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"json":"data"}`))
	}))
	badS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	cases := []struct {
		ds       *dataset.Dataset
		filename string
		fileLen  int
		err      string
	}{
		// bad input produces no result
		{&dataset.Dataset{}, "", 0, ""},

		// inline data
		{&dataset.Dataset{BodyBytes: []byte("a,b,c\n1,2,3")}, "", 0, "specifying bodyBytes requires format be specified in dataset.structure"},
		{&dataset.Dataset{Structure: &dataset.Structure{Format: "csv"}, BodyBytes: []byte("a,b,c\n1,2,3")}, "body.csv", 11, ""},

		// urlz
		{&dataset.Dataset{BodyPath: "http://"}, "", 0, "fetching body url: Get http:: http: no Host in request URL"},
		{&dataset.Dataset{BodyPath: fmt.Sprintf("%s/foobar.json", badS.URL)}, "", 0, "invalid status code fetching body url: 500"},
		{&dataset.Dataset{BodyPath: fmt.Sprintf("%s/foobar.json", s.URL)}, "foobar.json", 15, ""},

		// local filepaths
		{&dataset.Dataset{BodyPath: "nope.cbor"}, "", 0, "body file: open nope.cbor: no such file or directory"},
		{&dataset.Dataset{BodyPath: "nope.yaml"}, "", 0, "body file: open nope.yaml: no such file or directory"},
		{&dataset.Dataset{BodyPath: "testdata/schools.cbor"}, "schools.cbor", 154, ""},
		{&dataset.Dataset{BodyPath: "testdata/bad.yaml"}, "", 0, "converting yaml body to json: yaml: line 1: did not find expected '-' indicator"},
		{&dataset.Dataset{BodyPath: "testdata/oh_hai.yaml"}, "oh_hai.json", 29, ""},
	}

	for i, c := range cases {
		file, err := DatasetBodyFile(ctx, nil, c.ds)
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

func TestDatasetPinning(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)
	streams := ioes.NewDiscardIOStreams()

	if err := PinDataset(ctx, r, ref); err != nil {
		if err == repo.ErrNotPinner {
			t.Log("repo store doesn't support pinning")
		} else {
			t.Error(err.Error())
			return
		}
	}

	tc, err := dstest.NewTestCaseFromDir(testdataPath("counter"))
	if err != nil {
		t.Error(err.Error())
		return
	}

	ref2, err := CreateDataset(ctx, r, streams, tc.Input, nil, false, false, false, true)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if err := PinDataset(ctx, r, ref2); err != nil && err != repo.ErrNotPinner {
		// TODO (b5) - not sure what's going on here
		t.Log(err.Error())
		return
	}

	if err := UnpinDataset(ctx, r, ref); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}

	if err := UnpinDataset(ctx, r, ref2); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}
}

func TestConvertBodyFormat(t *testing.T) {
	jsonStructure := &dataset.Structure{Format: "json", Schema: dataset.BaseSchemaArray}
	csvStructure := &dataset.Structure{Format: "csv", Schema: dataset.BaseSchemaArray}

	// CSV -> JSON
	body := qfs.NewMemfileBytes("", []byte("a,b,c"))
	got, err := ConvertBodyFormat(body, csvStructure, jsonStructure)
	if err != nil {
		t.Error(err.Error())
	}
	data, err := ioutil.ReadAll(got)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(data, []byte(`[["a","b","c"]]`)) {
		t.Error(fmt.Errorf("converted body didn't match, got: %s", data))
	}

	// CSV -> JSON, multiple lines
	body = qfs.NewMemfileBytes("", []byte("a,b,c\n\rd,e,f\n\rg,h,i"))
	got, err = ConvertBodyFormat(body, csvStructure, jsonStructure)
	if err != nil {
		t.Fatal(err.Error())
	}
	data, err = ioutil.ReadAll(got)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(data, []byte(`[["a","b","c"],["d","e","f"],["g","h","i"]]`)) {
		t.Error(fmt.Errorf("converted body didn't match, got: %s", data))
	}

	// JSON -> CSV
	body = qfs.NewMemfileBytes("", []byte(`[["a","b","c"]]`))
	got, err = ConvertBodyFormat(body, jsonStructure, csvStructure)
	if err != nil {
		t.Fatal(err.Error())
	}
	data, err = ioutil.ReadAll(got)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(data, []byte("a,b,c\n")) {
		t.Error(fmt.Errorf("converted body didn't match, got: %s", data))
	}

	// CSV -> CSV
	body = qfs.NewMemfileBytes("", []byte("a,b,c"))
	got, err = ConvertBodyFormat(body, csvStructure, csvStructure)
	if err != nil {
		t.Fatal(err.Error())
	}
	data, err = ioutil.ReadAll(got)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(data, []byte("a,b,c\n")) {
		t.Error(fmt.Errorf("converted body didn't match, got: %s", data))
	}

	// JSON -> JSON
	body = qfs.NewMemfileBytes("", []byte(`[["a","b","c"]]`))
	got, err = ConvertBodyFormat(body, jsonStructure, jsonStructure)
	if err != nil {
		t.Fatal(err.Error())
	}
	data, err = ioutil.ReadAll(got)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(data, []byte(`[["a","b","c"]]`)) {
		t.Error(fmt.Errorf("converted body didn't match, got: %s", data))
	}
}
