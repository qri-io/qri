package core

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/qri-io/dataset"
	sql "github.com/qri-io/dataset_sql"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestList(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	req := NewQueryRequests(mr)
	if req == nil {
		t.Errorf("error: expected non-nil result from NewQueryRequests()")
		return
	}

	cases := []struct {
		p   *ListParams
		res *[]*repo.DatasetRef
		err string
	}{
		{&ListParams{"", 15, 1}, &[]*repo.DatasetRef{}, ""},
		{&ListParams{"", 50, 50}, &[]*repo.DatasetRef{}, ""},
	}
	for i, c := range cases {
		got := c.res
		err := req.List(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestGet(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	req := NewQueryRequests(mr)

	if req == nil {
		t.Errorf("error: expected non-nil result from NewQueryRequests()")
		return
	}
	cases := []struct {
		p   *GetQueryParams
		res *dataset.Dataset
		err string
	}{
		{&GetQueryParams{"", "", "", ""}, &dataset.Dataset{}, "error loading dataset: error getting file bytes: datastore: key not found"},
		// TODO: add more tests
		// {&GetQueryParams{"", "movies", "", ""}, &dataset.Dataset{}, ""},
	}
	for i, c := range cases {
		got := c.res
		err := req.Get(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestRun(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewQueryRequests(mr)

	if req == nil {
		t.Errorf("error: expected non-nil result from NewQueryRequests()")
		return
	}
	cases := []struct {
		p   *RunParams
		res *repo.DatasetRef
		err string
	}{
		{&RunParams{sql.ExecOpt{Format: dataset.CSVDataFormat}, "", nil}, &repo.DatasetRef{}, "dataset is required"},
		{&RunParams{sql.ExecOpt{Format: dataset.CSVDataFormat}, "", &dataset.Dataset{}}, &repo.DatasetRef{}, "error getting statement table names: syntax error at position 2"},
		{&RunParams{sql.ExecOpt{Format: dataset.CSVDataFormat}, "", &dataset.Dataset{QueryString: "select * from movies"}}, &repo.DatasetRef{}, ""},
		// TODO: add more tests

	}
	for i, c := range cases {
		got := c.res
		err := req.Run(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if c.err == "" {
			fmt.Println("path:", got.Path.String())
			df, err := mr.Store().Get(got.Path)
			if err != nil {
				t.Errorf("case %d error getting dataset path: %s: %s", i, got.Path.String(), err.Error())
				continue
			}

			ds := &dataset.Dataset{}
			if err := json.NewDecoder(df).Decode(ds); err != nil {
				t.Errorf("case %d decode dataset error: %s", i, err.Error())
				continue
			}

			if !ds.Transform.IsEmpty() {
				t.Errorf("expected stored dataset.Transform to be a reference")
			}
			if !ds.AbstractTransform.IsEmpty() {
				t.Errorf("expected stored dataset.AbstractTransform to be a reference")
			}
			if !ds.Structure.IsEmpty() {
				t.Errorf("expected stored dataset.Structure to be a reference")
			}
			if !ds.AbstractStructure.IsEmpty() {
				t.Errorf("expected stored dataset.AbstractStructure to be a reference")
			}

		}
	}
}

// TODO - RESTORE BEFORE MERGING
// func TestDatasetQueries(t *testing.T) {
// 	mr, err := testrepo.NewTestRepo()
// 	if err != nil {
// 		t.Errorf("error allocating test repo: %s", err.Error())
// 		return
// 	}

// 	req := NewQueryRequests(mr)

// 	path, err := mr.GetPath("movies")
// 	if err != nil {
// 		t.Errorf("errog getting path for 'movies' dataset: %s", err.Error())
// 		return
// 	}

// 	// ns, err := mr.Namespace(30, 0)
// 	// if err != nil {
// 	// 	t.Errorf("error getting repo namespace: %s", err.Error())
// 	// 	return
// 	// }

// 	// for _, n := range ns {
// 	// 	fmt.Println(n)
// 	// }

// 	qres := &repo.DatasetRef{}
// 	if err = req.Run(&RunParams{
// 		Dataset: &dataset.Dataset{
// 			QueryString: "select * from movies",
// 		}}, qres); err != nil {
// 		t.Errorf("error running query: %s", err.Error())
// 		return
// 	}

// 	cases := []struct {
// 		p   *DatasetQueriesParams
// 		res []*repo.DatasetRef
// 		err string
// 	}{
// 		{&DatasetQueriesParams{}, []*repo.DatasetRef{}, "path is required"},
// 		{&DatasetQueriesParams{Path: path.String()}, []*repo.DatasetRef{&repo.DatasetRef{}}, ""},
// 		// TODO: add more tests
// 	}

// 	for i, c := range cases {
// 		got := []*repo.DatasetRef{}
// 		err := req.DatasetQueries(c.p, &got)
// 		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
// 			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
// 			continue
// 		}

// 		// fmt.Println(got)

// 		if len(c.res) != len(got) {
// 			t.Errorf("case %d returned wrong number of responses. exepected: %d, got %d", i, len(c.res), len(got))
// 			continue
// 		}
// 	}
// }
