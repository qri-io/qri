package base

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/fs"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestListDatasets(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	res, err := ListDatasets(r, 1, 0, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 1 {
		t.Error("expected one dataset response")
	}

	res, err = ListDatasets(r, 1, 0, false, true)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 0 {
		t.Error("expected no published datasets")
	}

	if err := SetPublishStatus(r, &ref, true); err != nil {
		t.Fatal(err)
	}

	res, err = ListDatasets(r, 1, 0, false, true)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 1 {
		t.Error("expected one published dataset response")
	}
}

func TestCreateDataset(t *testing.T) {
	streams := ioes.NewDiscardIOStreams()
	r, err := repo.NewMemRepo(testPeerProfile, cafs.NewMapstore(), fs.NewMemFS(), profile.NewMemStore(), nil)
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
	ds.SetBodyFile(fs.NewMemfileBytes("body.json", []byte("[]")))

	if _, err := CreateDataset(r, streams, &dataset.Dataset{}, &dataset.Dataset{}, false, true); err == nil {
		t.Error("expected bad dataset to error")
	}

	ref, err := CreateDataset(r, streams, ds, &dataset.Dataset{}, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	refs, err := r.References(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Errorf("ref length mismatch. expected 1, got: %d", len(refs))
	}

	ds.Meta.Title = "an update"
	ds.PreviousPath = ref.Path
	ds.SetBodyFile(fs.NewMemfileBytes("body.json", []byte("[]")))

	prev := ref.Dataset

	ref, err = CreateDataset(r, streams, ds, prev, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	refs, err = r.References(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Errorf("ref length mismatch. expected 1, got: %d", len(refs))
	}
}

func TestFetchDataset(t *testing.T) {
	r1 := newTestRepo(t)
	r2 := newTestRepo(t)
	ref := addCitiesDataset(t, r2)

	// Connect in memory Mapstore's behind the scene to simulate IPFS-like behavior.
	r1.Store().(*cafs.MapStore).AddConnection(r2.Store().(*cafs.MapStore))

	if err := FetchDataset(r1, &repo.DatasetRef{Peername: "foo", Name: "bar"}, true, true); err == nil {
		t.Error("expected add of invalid ref to error")
	}

	if err := FetchDataset(r1, &ref, true, true); err != nil {
		t.Error(err.Error())
	}
}

func TestDatasetPodBodyFile(t *testing.T) {
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
		file, err := DatasetBodyFile(nil, c.ds)
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

// func TestDataset(t *testing.T) {
// 	rc, _ := mock.NewMockServer()

// 	rmf := func(t *testing.T) repo.Repo {
// 		mr, err := repo.NewMemRepo(testPeerProfile, cafs.NewMapstore(), profile.NewMemStore(), rc)
// 		if err != nil {
// 			panic(err)
// 		}
// 		// mr.SetPrivateKey(privKey)
// 		return mr
// 	}
// 	DatasetTests(t, rmf)
// }

// func TestSaveDataset(t *testing.T) {
// 	n := newTestNode(t)

// 	// test Dry run
// 	ds := &dataset.Dataset{
// 		Commit:    &dataset.Commit{},
// 		Structure: &dataset.Structure{Format: "json", Schema: dataset.BaseSchemaArray},
// 		Meta: &dataset.Meta{
// 			Title: "test title",
// 		},
// 	}
// 	body := fs.NewMemfileBytes("data.json", []byte("[]"))
// 	ref, _, err := SaveDataset(n, "dry_run_test", ds, body, nil, true, false)
// 	if err != nil {
// 		t.Errorf("dry run error: %s", err.Error())
// 	}
// 	if ref.AliasString() != "peer/dry_run_test" {
// 		t.Errorf("ref alias mismatch. expected: '%s' got: '%s'", "peer/dry_run_test", ref.AliasString())
// 	}
// }

// type RepoMakerFunc func(t *testing.T) repo.Repo
// type RepoTestFunc func(t *testing.T, rmf RepoMakerFunc)

// func DatasetTests(t *testing.T, rmf RepoMakerFunc) {
// 	for _, test := range []RepoTestFunc{
// 		testSaveDataset,
// 		testReadDataset,
// 		testRenameDataset,
// 		testDatasetPinning,
// 		testDeleteDataset,
// 		testEventsLog,
// 	} {
// 		test(t, rmf)
// 	}
// }

// func testSaveDataset(t *testing.T, rmf RepoMakerFunc) {
// 	createDataset(t, rmf)
// }

// func TestCreateDataset(t *testing.T, rmf RepoMakerFunc) (*p2p.QriNode, repo.DatasetRef) {
// 	r := rmf(t)
// 	r.SetProfile(testPeerProfile)
// 	n, err := p2p.NewQriNode(r, config.DefaultP2PForTesting())
// 	if err != nil {
// 		t.Error(err.Error())
// 		return n, repo.DatasetRef{}
// 	}

// 	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
// 	if err != nil {
// 		t.Error(err.Error())
// 		return n, repo.DatasetRef{}
// 	}

// 	ref, _, err := SaveDataset(n, tc.Name, tc.Input, tc.BodyFile(), nil, false, true)
// 	if err != nil {
// 		t.Error(err.Error())
// 	}

// 	return n, ref
// }

func TestReadDataset(t *testing.T) {
	// n, ref := createDataset(t, rmf)

	// if err := ReadDataset(n.Repo, &ref); err != nil {
	// 	t.Error(err.Error())
	// 	return
	// }

	// if ref.Dataset == nil {
	// 	t.Error("expected dataset to not equal nil")
	// 	return
	// }
}

// func testRenameDataset(t *testing.T, rmf RepoMakerFunc) {
// 	node, ref := createDataset(t, rmf)

// 	b := &repo.DatasetRef{
// 		Name:     "cities2",
// 		Peername: "me",
// 	}

// 	if err := RenameDataset(node, &ref, b); err != nil {
// 		t.Error(err.Error())
// 		return
// 	}

// 	if err := ReadDataset(node.Repo, b); err != nil {
// 		t.Error(err.Error())
// 		return
// 	}

// 	if b.Dataset == nil {
// 		t.Error("expected dataset to not equal nil")
// 		return
// 	}
// }

func TestDatasetPinning(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)
	streams := ioes.NewDiscardIOStreams()

	if err := PinDataset(r, ref); err != nil {
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

	ref2, err := CreateDataset(r, streams, tc.Input, nil, false, false)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if err := PinDataset(r, ref2); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}

	if err := UnpinDataset(r, ref); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}

	if err := UnpinDataset(r, ref2); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}
}

// func testDeleteDataset(t *testing.T, rmf RepoMakerFunc) {
// 	node, ref := createDataset(t, rmf)

// 	if err := DeleteDataset(node, &ref); err != nil {
// 		t.Error(err.Error())
// 		return
// 	}
// }

// func testEventsLog(t *testing.T, rmf RepoMakerFunc) {
// 	node, ref := createDataset(t, rmf)
// 	pinner := true

// 	b := &repo.DatasetRef{
// 		Name:      "cities2",
// 		ProfileID: ref.ProfileID,
// 	}

// 	if err := RenameDataset(node, &ref, b); err != nil {
// 		t.Error(err.Error())
// 		return
// 	}

// 	if err := PinDataset(node.Repo, *b); err != nil {
// 		if err == repo.ErrNotPinner {
// 			pinner = false
// 		} else {
// 			t.Error(err.Error())
// 			return
// 		}
// 	}

// 	// TODO - calling unpin followed by delete will trigger two unpin events,
// 	// which based on our current architecture can and will probably cause problems
// 	// we should either hardern every unpin implementation to not error on multiple
// 	// calls to unpin the same hash, or include checks in the delete method
// 	// and only call unpin if the hash is in fact pinned
// 	// if err := act.UnpinDataset(b); err != nil && err != repo.ErrNotPinner {
// 	// 	t.Error(err.Error())
// 	// 	return
// 	// }

// 	if err := DeleteDataset(node, b); err != nil {
// 		t.Error(err.Error())
// 		return
// 	}

// 	events, err := node.Repo.Events(10, 0)
// 	if err != nil {
// 		t.Error(err.Error())
// 		return
// 	}

// 	ets := []repo.EventType{repo.ETDsDeleted, repo.ETDsUnpinned, repo.ETDsPinned, repo.ETDsRenamed, repo.ETDsPinned, repo.ETDsCreated}

// 	if !pinner {
// 		ets = []repo.EventType{repo.ETDsDeleted, repo.ETDsRenamed, repo.ETDsCreated}
// 	}

// 	if len(events) != len(ets) {
// 		t.Errorf("event log length mismatch. expected: %d, got: %d", len(ets), len(events))
// 		t.Log("event log:")
// 		for i, e := range events {
// 			t.Logf("\t%d: %s", i, e.Type)
// 		}
// 		return
// 	}

// 	for i, et := range ets {
// 		if events[i].Type != et {
// 			t.Errorf("case %d eventType mismatch. expected: %s, got: %s", i, et, events[i].Type)
// 		}
// 	}
// }

func TestConvertBodyFormat(t *testing.T) {
	jsonStructure := &dataset.Structure{Format: "json", Schema: dataset.BaseSchemaArray}
	csvStructure := &dataset.Structure{Format: "csv", Schema: dataset.BaseSchemaArray}

	// CSV -> JSON
	body := fs.NewMemfileBytes("", []byte("a,b,c"))
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
	body = fs.NewMemfileBytes("", []byte("a,b,c\n\rd,e,f\n\rg,h,i"))
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
	body = fs.NewMemfileBytes("", []byte(`[["a","b","c"]]`))
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
	body = fs.NewMemfileBytes("", []byte("a,b,c"))
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
	body = fs.NewMemfileBytes("", []byte(`[["a","b","c"]]`))
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
