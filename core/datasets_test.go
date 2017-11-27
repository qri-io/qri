package core

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestDatasetRequestsInit(t *testing.T) {
	badDataFile := testrepo.BadDataFile
	jobsByAutomationFile := testrepo.JobsByAutomationFile
	jobsByAutomationFile2 := testrepo.JobsByAutomationFile2
	badDataFormatFile := testrepo.BadDataFormatFile
	badStructureFile := testrepo.BadStructureFile

	cases := []struct {
		p   *InitDatasetParams
		res *repo.DatasetRef
		err string
	}{
		{&InitDatasetParams{}, nil, "either a file or a url is required to create a dataset"},
		{&InitDatasetParams{Data: badDataFile}, nil, "error detecting format extension: no file extension provided"},
		{&InitDatasetParams{DataFilename: badDataFile.FileName(), Data: badDataFile}, nil, "invalid data format: error reading first row of csv: EOF"},
		// Ensure that DataFormat validation is being called
		{&InitDatasetParams{DataFilename: badDataFormatFile.FileName(),
			Data: badDataFormatFile}, nil, "invalid data format: error: inconsistent column length on line 2 of length 3 (rather than 4). ensure all csv columns same length"},
		// Ensure that structure validation is being called
		{&InitDatasetParams{DataFilename: badStructureFile.FileName(),
			Data: badStructureFile}, nil, "invalid structure: error: cannot use the same name, 'colb' more than once"},
		// this should work
		{&InitDatasetParams{DataFilename: jobsByAutomationFile.FileName(), Data: jobsByAutomationFile}, nil, ""},
		// Ensure that we can't double-add data
		{&InitDatasetParams{DataFilename: jobsByAutomationFile2.FileName(), Data: jobsByAutomationFile2}, nil, "this data already exists"},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewDatasetRequests(mr)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.InitDataset(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestDatasetRequestsList(t *testing.T) {
	var (
		movies, counter, cities, archive *repo.DatasetRef
	)

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	refs, err := mr.Namespace(30, 0)
	if err != nil {
		t.Errorf("error getting namespace: %s", err.Error())
		return
	}

	for _, ref := range refs {
		switch ref.Name {
		case "movies":
			movies = ref
		case "counter":
			counter = ref
		case "cities":
			cities = ref
		case "archive":
			archive = ref
		}
	}

	cases := []struct {
		p   *ListParams
		res []*repo.DatasetRef
		err string
	}{
		{&ListParams{OrderBy: "", Limit: 1, Offset: 0}, nil, ""},
		{&ListParams{OrderBy: "chaos", Limit: 1, Offset: -50}, nil, ""},
		{&ListParams{OrderBy: "", Limit: 30, Offset: 0}, []*repo.DatasetRef{archive, cities, counter, movies}, ""},
		{&ListParams{OrderBy: "timestamp", Limit: 30, Offset: 0}, []*repo.DatasetRef{archive, cities, counter, movies}, ""},
		// TODO: re-enable {&ListParams{OrderBy: "name", Limit: 30, Offset: 0}, []*repo.DatasetRef{cities, counter, movies}, ""},
	}

	req := NewDatasetRequests(mr)
	for i, c := range cases {
		got := []*repo.DatasetRef{}
		err := req.List(c.p, &got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if c.err == "" && c.res != nil {
			if len(c.res) != len(got) {
				t.Errorf("case %d response length mismatch. expected %d, got: %d", i, len(c.res), len(got))
				continue
			}
			for j, expect := range c.res {
				if err := repo.CompareDatasetRef(expect, got[j]); err != nil {
					t.Errorf("case %d expected dataset error. index %d mismatch: %s", i, j, err.Error())
					continue
				}
			}
		}
	}
}

func TestDatasetRequestsGet(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	path, err := mr.GetPath("movies")
	if err != nil {
		t.Errorf("error getting path: %s", err.Error())
		return
	}
	moviesDs, err := dsfs.LoadDataset(mr.Store(), path)
	if err != nil {
		t.Errorf("error loading dataset: %s", err.Error())
		return
	}
	cases := []struct {
		p   *GetDatasetParams
		res *dataset.Dataset
		err string
	}{
		//TODO: probably delete some of these
		{&GetDatasetParams{Path: datastore.NewKey("abc"), Name: "ABC", Hash: "123"}, nil, "error loading dataset: error getting file bytes: datastore: key not found"},
		{&GetDatasetParams{Path: path, Name: "ABC", Hash: "123"}, nil, ""},
		{&GetDatasetParams{Path: path, Name: "movies", Hash: "123"}, moviesDs, ""},
		{&GetDatasetParams{Path: path, Name: "cats", Hash: "123"}, moviesDs, ""},
	}

	req := NewDatasetRequests(mr)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Get(c.p, got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
		// if got != c.res && c.checkResult == true {
		// 	t.Errorf("case %d result mismatch: \nexpected \n\t%s, \n\ngot: \n%s", i, c.res, got)
		// }
	}
}

func TestDatasetRequestsUpdate(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	// path, err := mr.GetPath("movies")
	// if err != nil {
	// 	t.Errorf("error getting path: %s", err.Error())
	// 	return
	// }
	// moviesDs, err := dsfs.LoadDataset(mr.Store(), path)
	// if err != nil {
	// 	t.Errorf("error loading dataset: %s", err.Error())
	// 	return
	// }
	cases := []struct {
		p   *UpdateParams
		res *repo.DatasetRef
		err string
	}{
	//TODO: probably delete some of these
	// {&UpdateParams{Path: datastore.NewKey("abc"), Name: "ABC", Hash: "123"}, nil, "error loading dataset: error getting file bytes: datastore: key not found"},
	// {&UpdateParams{Path: path, Name: "ABC", Hash: "123"}, nil, ""},
	// {&UpdateParams{Path: path, Name: "movies", Hash: "123"}, moviesDs, ""},
	// {&UpdateParams{Path: path, Name: "cats", Hash: "123"}, moviesDs, ""},
	}

	req := NewDatasetRequests(mr)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Update(c.p, got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
		// if got != c.res && c.checkResult == true {
		// 	t.Errorf("case %d result mismatch: \nexpected \n\t%s, \n\ngot: \n%s", i, c.res, got)
		// }
	}
}

func TestDatasetRequestsDelete(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	path, err := mr.GetPath("movies")
	if err != nil {
		t.Errorf("error getting path: %s", err.Error())
		return
	}

	cases := []struct {
		p   *DeleteParams
		res *dataset.Dataset
		err string
	}{
		{&DeleteParams{}, nil, "either name or path is required"},
		{&DeleteParams{Path: datastore.NewKey("abc"), Name: "ABC"}, nil, "repo: not found"},
		{&DeleteParams{Path: path}, nil, ""},
	}

	req := NewDatasetRequests(mr)
	for i, c := range cases {
		got := false
		err := req.Delete(c.p, &got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestDatasetRequestsStructuredData(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	moviesPath, err := mr.GetPath("movies")
	if err != nil {
		t.Errorf("error getting movies path: %s", err.Error())
		return
	}
	archivePath, err := mr.GetPath("archive")
	if err != nil {
		t.Errorf("error getting archive path: %s", err.Error())
		return
	}
	var df1 = dataset.JsonDataFormat
	cases := []struct {
		p        *StructuredDataParams
		resCount int
		err      string
	}{
		{&StructuredDataParams{}, 0, "error getting file bytes: datastore: key not found"},
		{&StructuredDataParams{Format: df1, Path: moviesPath, Limit: 5, Offset: 0, All: false}, 5, ""},
		{&StructuredDataParams{Format: df1, Path: moviesPath, Limit: -5, Offset: -100, All: false}, 0, ""},
		{&StructuredDataParams{Format: df1, Path: moviesPath, Limit: -5, Offset: -100, All: true}, 0, ""},
		{&StructuredDataParams{Format: df1, Path: moviesPath, Limit: -5, Offset: -100, All: true}, 0, ""},
		{&StructuredDataParams{Format: df1, Path: moviesPath, Limit: -5, Offset: -100, All: true}, 0, ""},
		{&StructuredDataParams{Format: df1, Path: moviesPath, Limit: -5, Offset: -100, All: true}, 0, ""},
		{&StructuredDataParams{Format: dataset.JsonDataFormat, Path: archivePath, Limit: 0, Offset: 0, All: true}, 0, ""},
	}

	req := NewDatasetRequests(mr)
	for i, c := range cases {
		got := &StructuredData{}
		err := req.StructuredData(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if got.Data == nil && c.resCount == 0 {
			continue
		}

		switch c.p.Format {
		default:
			// default should be json format
			_, err := json.Marshal(got.Data)
			if err != nil {
				t.Errorf("case %d error parsing response data: %s", i, err.Error())
				continue
			}
		case dataset.CsvDataFormat:
			r := csv.NewReader(bytes.NewBuffer(got.Data.([]byte)))
			_, err := r.ReadAll()
			if err != nil {
				t.Errorf("case %d error parsing response data: %s", err.Error())
				continue
			}
		}
	}
}

func TestDatasetRequestsAddDataset(t *testing.T) {
	cases := []struct {
		p   *AddParams
		res *repo.DatasetRef
		err string
	}{
		{&AddParams{Name: "abc", Hash: "hash###"}, nil, "can only add datasets when running an IPFS filestore"},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewDatasetRequests(mr)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.AddDataset(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}
