package core

import (
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

	cases := []struct {
		p   *InitDatasetParams
		res *repo.DatasetRef
		err string
	}{
		{&InitDatasetParams{}, nil, "either a file or a url is required to create a dataset"},
		{&InitDatasetParams{Data: badDataFile}, nil, "error determining dataset schema: no file extension provided"},
		{&InitDatasetParams{DataFilename: badDataFile.FileName(), Data: badDataFile}, nil, "error determining dataset schema: EOF"},
		{&InitDatasetParams{DataFilename: jobsByAutomationFile.FileName(), Data: jobsByAutomationFile}, nil, ""},
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
		movies, counter, cities *repo.DatasetRef
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
		}
	}

	cases := []struct {
		p   *ListParams
		res []*repo.DatasetRef
		err string
	}{
		{&ListParams{OrderBy: "", Limit: 1, Offset: 0}, nil, ""},
		{&ListParams{OrderBy: "chaos", Limit: 1, Offset: -50}, nil, ""},
		{&ListParams{OrderBy: "", Limit: 30, Offset: 0}, []*repo.DatasetRef{movies, counter, cities}, ""},
		{&ListParams{OrderBy: "timestamp", Limit: 30, Offset: 0}, []*repo.DatasetRef{movies, counter, cities}, ""},
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
	path, err := mr.GetPath("movies")
	if err != nil {
		t.Errorf("error getting path: %s", err.Error())
		return
	}
	var df1 dataset.DataFormat = 0
	cases := []struct {
		p   *StructuredDataParams
		res *StructuredData
		err string
	}{
		{&StructuredDataParams{}, nil, "error getting file bytes: datastore: key not found"},
		{&StructuredDataParams{Format: df1, Path: path, Objects: false, Limit: 5, Offset: 0, All: false}, nil, ""},
		{&StructuredDataParams{Format: df1, Path: path, Objects: false, Limit: -5, Offset: -100, All: false}, nil, ""},
		{&StructuredDataParams{Format: df1, Path: path, Objects: false, Limit: -5, Offset: -100, All: true}, nil, ""},
	}

	req := NewDatasetRequests(mr)
	for i, c := range cases {
		got := &StructuredData{}
		err := req.StructuredData(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
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
