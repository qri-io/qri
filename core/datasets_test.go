package core

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dsdiff"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestDatasetRequestsInit(t *testing.T) {
	badDataFile := testrepo.BadDataFile
	jobsByAutomationFile := testrepo.JobsByAutomationFile
	// badDataFormatFile := testrepo.BadDataFormatFile
	// badStructureFile := testrepo.BadStructureFile

	cases := []struct {
		p   *InitParams
		res *repo.DatasetRef
		err string
	}{
		{&InitParams{}, nil, "either a file or a url is required to create a dataset"},
		{&InitParams{Data: badDataFile}, nil, "error determining dataset schema: no file extension provided"},
		{&InitParams{DataFilename: badDataFile.FileName(), Data: badDataFile}, nil, "error determining dataset schema: EOF"},
		// TODO - reenable
		// Ensure that DataFormat validation is being called
		// {&InitParams{DataFilename: badDataFormatFile.FileName(),
		// Data: badDataFormatFile}, nil, "invalid data format: error: inconsistent column length on line 2 of length 3 (rather than 4). ensure all csv columns same length"},
		// TODO - restore
		// Ensure that structure validation is being called
		// {&InitParams{DataFilename: badStructureFile.FileName(),
		// 	Data: badStructureFile}, nil, "invalid structure: schema: fields: error: cannot use the same name, 'col_b' more than once"},
		// should reject invalid names
		{&InitParams{DataFilename: jobsByAutomationFile.FileName(), Peername: "peer", Name: "foo bar baz", Data: jobsByAutomationFile}, nil,
			"invalid name: error: illegal name 'foo bar baz', names must start with a letter and consist of only a-z,0-9, and _. max length 144 characters"},
		// this should work
		{&InitParams{DataFilename: jobsByAutomationFile.FileName(), Peername: "peer", Data: jobsByAutomationFile}, nil, ""},
		// Ensure that we can't double-add data
		// {&InitParams{DataFilename: jobsByAutomationFile2.FileName(), Data: jobsByAutomationFile2}, nil, "this data already exists"},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewDatasetRequests(mr, nil)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Init(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestDatasetRequestsList(t *testing.T) {
	var (
		movies, counter, cities, craigslist repo.DatasetRef
	)

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	refs, err := mr.References(30, 0)
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
		case "craigslist":
			craigslist = ref
		}
	}

	cases := []struct {
		p   *ListParams
		res []repo.DatasetRef
		err string
	}{
		{&ListParams{OrderBy: "", Limit: 1, Offset: 0}, nil, ""},
		{&ListParams{OrderBy: "chaos", Limit: 1, Offset: -50}, nil, ""},
		{&ListParams{OrderBy: "", Limit: 30, Offset: 0}, []repo.DatasetRef{cities, counter, craigslist, movies}, ""},
		{&ListParams{OrderBy: "timestamp", Limit: 30, Offset: 0}, []repo.DatasetRef{cities, counter, craigslist, movies}, ""},
		{&ListParams{Peername: "me", OrderBy: "timestamp", Limit: 30, Offset: 0}, []repo.DatasetRef{cities, counter, craigslist, movies}, ""},
		// TODO: re-enable {&ListParams{OrderBy: "name", Limit: 30, Offset: 0}, []*repo.DatasetRef{cities, counter, movies}, ""},
	}

	req := NewDatasetRequests(mr, nil)
	for i, c := range cases {
		got := []repo.DatasetRef{}
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
	ref, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Errorf("error getting path: %s", err.Error())
		return
	}

	moviesDs, err := dsfs.LoadDataset(mr.Store(), datastore.NewKey(ref.Path))
	if err != nil {
		t.Errorf("error loading dataset: %s", err.Error())
		return
	}

	cases := []struct {
		p   repo.DatasetRef
		res *dataset.Dataset
		err string
	}{
		//TODO: probably delete some of these
		{repo.DatasetRef{Peername: "peer", Path: "abc", Name: "ABC"}, nil, "error loading dataset: error getting file bytes: datastore: key not found"},
		{repo.DatasetRef{Peername: "peer", Path: ref.Path, Name: "ABC"}, nil, ""},
		{repo.DatasetRef{Peername: "peer", Path: ref.Path, Name: "movies"}, moviesDs, ""},
		{repo.DatasetRef{Peername: "peer", Path: ref.Path, Name: "cats"}, moviesDs, ""},
	}

	req := NewDatasetRequests(mr, nil)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Get(&c.p, got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
		// if got != c.res && c.checkResult == true {
		// 	t.Errorf("case %d result mismatch: \nexpected \n\t%s, \n\ngot: \n%s", i, c.res, got)
		// }
	}
}

func TestDatasetRequestsSave(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	cases := []struct {
		p   *SaveParams
		err string
	}{
		//TODO: probably delete some of these
		// {&SaveParams{Path: datastore.NewKey("abc"), Name: "ABC", Hash: "123"}, nil, "error loading dataset: error getting file bytes: datastore: key not found"},
		// {&SaveParams{Path: path, Name: "ABC", Hash: "123"}, nil, ""},
		{&SaveParams{Name: "movies", Peername: "peer", MetadataFilename: "meta.json", Metadata: bytes.NewReader([]byte(`{"title":"movies!"}`))}, ""},
		{&SaveParams{Name: "unknown_dataset", Peername: "peer"}, "error getting previous dataset: error loading dataset: error getting file bytes: datastore: key not found"},
		// {&SaveParams{Path: path, Name: "cats"}, moviesDs, ""},
	}

	req := NewDatasetRequests(mr, nil)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Save(c.p, got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
		// if got != c.res && c.checkResult == true {
		// 	t.Errorf("case %d result mismatch: \nexpected \n\t%s, \n\ngot: \n%s", i, c.res, got)
		// }
	}
}

func TestDatasetRequestsRename(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	cases := []struct {
		p   *RenameParams
		res string
		err string
	}{
		{&RenameParams{}, "", "current name is required to rename a dataset"},
		{&RenameParams{Current: repo.DatasetRef{Peername: "peer", Name: "movies"}, New: repo.DatasetRef{Peername: "peer", Name: "new movies"}}, "", "error: illegal name 'new movies', names must start with a letter and consist of only a-z,0-9, and _. max length 144 characters"},
		{&RenameParams{Current: repo.DatasetRef{Peername: "peer", Name: "movies"}, New: repo.DatasetRef{Peername: "peer", Name: "new_movies"}}, "new_movies", ""},
		{&RenameParams{Current: repo.DatasetRef{Peername: "peer", Name: "new_movies"}, New: repo.DatasetRef{Peername: "peer", Name: "new_movies"}}, "", "dataset 'peer/new_movies' already exists"},
	}

	req := NewDatasetRequests(mr, nil)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Rename(c.p, got)
		t.Log(mr.References(30, 0))

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if got.Name != c.res {
			t.Errorf("case %d response name mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}
	}
}

func TestDatasetRequestsRemove(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	ref, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Errorf("error getting movies ref: %s", err.Error())
		return
	}

	cases := []struct {
		p   *repo.DatasetRef
		res *dataset.Dataset
		err string
	}{
		{&repo.DatasetRef{}, nil, "either peername/name or path is required"},
		{&repo.DatasetRef{Path: "abc", Name: "ABC"}, nil, "repo: not found"},
		{&ref, nil, ""},
	}

	req := NewDatasetRequests(mr, nil)
	for i, c := range cases {
		got := false
		err := req.Remove(c.p, &got)

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
	moviesRef, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Errorf("error getting movies ref: %s", err.Error())
		return
	}
	clRef, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "craigslist"})
	if err != nil {
		t.Errorf("error getting craigslist ref: %s", err.Error())
		return
	}
	var df1 = dataset.JSONDataFormat
	cases := []struct {
		p        *StructuredDataParams
		resCount int
		err      string
	}{
		{&StructuredDataParams{}, 0, "error loading dataset: error getting file bytes: datastore: key not found"},
		{&StructuredDataParams{Format: df1, Path: moviesRef.Path, Limit: 5, Offset: 0, All: false}, 5, ""},
		{&StructuredDataParams{Format: df1, Path: moviesRef.Path, Limit: -5, Offset: -100, All: false}, 0, "invalid limit / offset settings"},
		{&StructuredDataParams{Format: df1, Path: moviesRef.Path, Limit: -5, Offset: -100, All: true}, 0, "invalid limit / offset settings"},
		{&StructuredDataParams{Format: df1, Path: clRef.Path, Limit: 0, Offset: 0, All: true}, 0, ""},
	}

	req := NewDatasetRequests(mr, nil)
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
		case dataset.CSVDataFormat:
			r := csv.NewReader(bytes.NewBuffer(got.Data))
			_, err := r.ReadAll()
			if err != nil {
				t.Errorf("case %d error parsing response data: %s", i, err.Error())
				continue
			}
		}
	}
}

func TestDatasetRequestsAdd(t *testing.T) {
	cases := []struct {
		p   *repo.DatasetRef
		res *repo.DatasetRef
		err string
	}{
		{&repo.DatasetRef{Name: "abc", Path: "hash###"}, nil, "can only add datasets when running an IPFS filestore"},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewDatasetRequests(mr, nil)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Add(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestDatasetRequestsValidate(t *testing.T) {
	movieb := []byte(`movie_title,duration
Avatar ,178
Pirates of the Caribbean: At World's End ,169
Pirates of the Caribbean: At World's End ,foo
`)
	schemaB := []byte(`{
	  "type": "array",
	  "items": {
	    "type": "array",
	    "items": [
	      {
	        "title": "title",
	        "type": "string"
	      },
	      {
	        "title": "duration",
	        "type": "number"
	      }
	    ]
	  }
	}`)

	dataf := cafs.NewMemfileBytes("data.csv", movieb)
	dataf2 := cafs.NewMemfileBytes("data.csv", movieb)
	schemaf := cafs.NewMemfileBytes("schema.json", schemaB)
	schemaf2 := cafs.NewMemfileBytes("schema.json", schemaB)

	cases := []struct {
		p         ValidateDatasetParams
		numErrors int
		err       string
	}{
		{ValidateDatasetParams{Ref: repo.DatasetRef{}}, 0, "either data or a dataset reference is required"},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me"}}, 0, "cannot find dataset: peer@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me", Name: "movies"}}, 15, ""},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me", Name: "movies"}, Data: dataf, DataFilename: "data.csv"}, 1, ""},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me", Name: "movies"}, Schema: schemaf}, 15, ""},
		{ValidateDatasetParams{Schema: schemaf2, DataFilename: "data.csv", Data: dataf2}, 1, ""},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewDatasetRequests(mr, nil)
	for i, c := range cases {
		got := []jsonschema.ValError{}
		err := req.Validate(&c.p, &got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err.Error())
			continue
		}

		if len(got) != c.numErrors {
			t.Errorf("case %d error count mismatch. expected: %d, got: %d", i, c.numErrors, len(got))
			t.Log(got)
			continue
		}
	}
}

func TestDataRequestsDiff(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	req := NewDatasetRequests(mr, nil)
	// File 1
	dsRef1 := &repo.DatasetRef{}
	initParams := &InitParams{
		Peername:     "peer",
		DataFilename: jobsByAutomationFile.FileName(),
		Data:         jobsByAutomationFile,
		// MetadataFilename: jobsMeta.FileName(),
		// Metadata:         jobsMeta,
	}
	err = req.Init(initParams, dsRef1)
	if err != nil {
		t.Errorf("couldn't load file 1: %s", err.Error())
		return
	}
	dsBase, err := dsfs.LoadDataset(mr.Store(), datastore.NewKey(dsRef1.Path))
	if err != nil {
		t.Errorf("error loading dataset 1: %s", err.Error())
		return
	}
	// File 2
	dsRef2 := &repo.DatasetRef{}
	initParams = &InitParams{
		Peername:     "peer",
		DataFilename: jobsByAutomationFile2.FileName(),
		Data:         jobsByAutomationFile2,
	}
	err = req.Init(initParams, dsRef2)
	if err != nil {
		t.Errorf("couldn't load second file: %s", err.Error())
		return
	}
	dsNewStructure, err := dsfs.LoadDataset(mr.Store(), datastore.NewKey(dsRef2.Path))
	if err != nil {
		t.Errorf("error loading dataset: %s", err.Error())
		return
	}

	//test cases
	cases := []struct {
		dsLeft, dsRight *dataset.Dataset
		displayFormat   string
		expected        string
		err             string
	}{
		{dsBase, dsNewStructure, "listKeys", "Structure: 3 changes\n\t- modified checksum\n\t- modified length\n\t- modified schema", ""},
	}
	// execute
	for i, c := range cases {
		got, err := dsdiff.DiffDatasets(c.dsLeft, c.dsRight, nil)
		if err != nil {
			if err.Error() == c.err {
				continue
			} else {
				t.Errorf("case %d error mismatch: expected '%s', got '%s'", i, c.err, err.Error())
				return
			}
		}
		stringDiffs, err := dsdiff.MapDiffsToString(got, c.displayFormat)
		if err != nil {
			t.Errorf("case %d error mapping to string: %s", i, err.Error())
		}
		if stringDiffs != c.expected {
			t.Errorf("case %d response mistmatch: expected '%s', got '%s'", i, c.expected, stringDiffs)
		}
	}
}

var jobsByAutomationFile = cafs.NewMemfileBytes("jobs_ranked_by_automation_probability.csv", []byte(`rank,probability_of_automation,soc_code,job_title
702,"0.99","41-9041","Telemarketers"
701,"0.99","23-2093","Title Examiners, Abstractors, and Searchers"
700,"0.99","51-6051","Sewers, Hand"
699,"0.99","15-2091","Mathematical Technicians"
698,"0.99","13-2053","Insurance Underwriters"
697,"0.99","49-9064","Watch Repairers"
696,"0.99","43-5011","Cargo and Freight Agents"
695,"0.99","13-2082","Tax Preparers"
694,"0.99","51-9151","Photographic Process Workers and Processing Machine Operators"
693,"0.99","43-4141","New Accounts Clerks"
692,"0.99","25-4031","Library Technicians"
691,"0.99","43-9021","Data Entry Keyers"
690,"0.98","51-2093","Timing Device Assemblers and Adjusters"
689,"0.98","43-9041","Insurance Claims and Policy Processing Clerks"
688,"0.98","43-4011","Brokerage Clerks"
687,"0.98","43-4151","Order Clerks"
686,"0.98","13-2072","Loan Officers"
685,"0.98","13-1032","Insurance Appraisers, Auto Damage"
684,"0.98","27-2023","Umpires, Referees, and Other Sports Officials"
683,"0.98","43-3071","Tellers"
682,"0.98","51-9194","Etchers and Engravers"
681,"0.98","51-9111","Packaging and Filling Machine Operators and Tenders"
680,"0.98","43-3061","Procurement Clerks"
679,"0.98","43-5071","Shipping, Receiving, and Traffic Clerks"
678,"0.98","51-4035","Milling and Planing Machine Setters, Operators, and Tenders, Metal and Plastic"
677,"0.98","13-2041","Credit Analysts"
676,"0.98","41-2022","Parts Salespersons"
675,"0.98","13-1031","Claims Adjusters, Examiners, and Investigators"
674,"0.98","53-3031","Driver/Sales Workers"
673,"0.98","27-4013","Radio Operators"
`))

var jobsByAutomationFile2 = cafs.NewMemfileBytes("jobs_ranked_by_automation_prob.csv", []byte(`rank,probability_of_automation,industry_code,job_name
702,"0.99","41-9041","Telemarketers"
701,"0.99","23-2093","Title Examiners, Abstractors, and Searchers"
700,"0.99","51-6051","Sewers, Hand"
699,"0.99","15-2091","Mathematical Technicians"
698,"0.88","13-2053","Insurance Underwriters"
697,"0.99","49-9064","Watch Repairers"
696,"0.99","43-5011","Cargo and Freight Agents"
695,"0.99","13-2082","Tax Preparers"
694,"0.99","51-9151","Photographic Process Workers and Processing Machine Operators"
693,"0.99","43-4141","New Accounts Clerks"
692,"0.99","25-4031","Library Technicians"
691,"0.99","43-9021","Data Entry Keyers"
690,"0.98","51-2093","Timing Device Assemblers and Adjusters"
689,"0.98","43-9041","Insurance Claims and Policy Processing Clerks"
688,"0.98","43-4011","Brokerage Clerks"
687,"0.98","43-4151","Order Clerks"
686,"0.98","13-2072","Loan Officers"
685,"0.98","13-1032","Insurance Appraisers, Auto Damage"
684,"0.98","27-2023","Umpires, Referees, and Other Sports Officials"
683,"0.98","43-3071","Tellers"
682,"0.98","51-9194","Etchers and Engravers"
681,"0.98","51-9111","Packaging and Filling Machine Operators and Tenders"
680,"0.98","43-3061","Procurement Clerks"
679,"0.98","43-5071","Shipping, Receiving, and Traffic Clerks"
678,"0.98","51-4035","Milling and Planing Machine Setters, Operators, and Tenders, Metal and Plastic"
677,"0.98","13-2041","Credit Analysts"
676,"0.98","41-2022","Parts Salespersons"
675,"0.98","13-1031","Claims Adjusters, Examiners, and Investigators"
674,"0.98","53-3031","Driver/Sales Workers"
673,"0.98","27-4013","Radio Operators"
`))
