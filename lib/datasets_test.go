package lib

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	cmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	reporef "github.com/qri-io/qri/repo/ref"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestDatasetRequestsSave(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	jobsBodyPath, err := dstest.BodyFilepath("testdata/jobs_by_automation")
	if err != nil {
		t.Fatal(err.Error())
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := `city,pop,avg_age,in_usa
	toronto,40000000,55.5,false
	new york,8500000,44.4,true
	chicago,300000,44.4,true
	chatham,35000,65.25,true
	raleigh,250000,50.65,true
	sarnia,550000,55.65,false
`
		w.Write([]byte(res))
	}))

	badDataS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`\\\{"json":"data"}`))
	}))

	citiesMetaOnePath := tempDatasetFile(t, "*-cities_meta_1.json", &dataset.Dataset{Meta: &dataset.Meta{Title: "updated name of movies dataset"}})
	citiesMetaTwoPath := tempDatasetFile(t, "*-cities_meta_2.json", &dataset.Dataset{Meta: &dataset.Meta{Description: "Description, b/c bodies are the same thing"}})
	defer func() {
		os.RemoveAll(citiesMetaOnePath)
		os.RemoveAll(citiesMetaTwoPath)
	}()

	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	privateErrMsg := "option to make dataset private not yet implemented, refer to https://github.com/qri-io/qri/issues/291 for updates"
	_, err = inst.Dataset().Save(ctx, &SaveParams{Private: true})
	if err == nil {
		t.Errorf("expected datset to error")
	} else if err.Error() != privateErrMsg {
		t.Errorf("private flag error mismatch: expected: '%s', got: '%s'", privateErrMsg, err.Error())
	}

	good := []struct {
		description string
		params      SaveParams
		res         *reporef.DatasetRef
	}{
		{"body file", SaveParams{Ref: "me/jobs_ranked_by_automation_prob", BodyPath: jobsBodyPath}, nil},
		{"no body", SaveParams{Ref: "me/no_body_dataset", Dataset: &dataset.Dataset{Meta: &dataset.Meta{Title: "big things cooking"}}}, nil},
		{"meta set title", SaveParams{Ref: "me/cities", FilePaths: []string{citiesMetaOnePath}}, nil},
		{"meta set description, supply same body", SaveParams{Ref: "me/cities", FilePaths: []string{citiesMetaTwoPath}, BodyPath: s.URL + "/body.csv"}, nil},
	}

	for i, c := range good {
		got, err := inst.Dataset().Save(ctx, &c.params)
		if err != nil {
			t.Errorf("case %d: '%s' unexpected error: %s", i, c.description, err.Error())
			continue
		}

		if got != nil && c.res != nil {
			expect := c.res.Dataset
			if diff := dstest.CompareDatasets(expect, got); diff != "" {
				t.Errorf("case %d ds mistmatch (-want +got):\n%s", i, diff)
				continue
			}
		}
	}

	bad := []struct {
		description string
		params      SaveParams
		err         string
	}{

		{"empty params", SaveParams{}, "no changes to save"},
		{"", SaveParams{Ref: "me/bad", BodyPath: badDataS.URL + "/data.json"}, "determining dataset structure: invalid json data"},
	}

	for i, c := range bad {
		_, err := inst.Dataset().Save(ctx, &c.params)
		if err == nil {
			t.Errorf("case %d: '%s' returned no error", i, c.description)
		}
		if err.Error() != c.err {
			t.Errorf("case %d: '%s' error mismatch. expected:\n'%s'\ngot:\n'%s'", i, c.description, c.err, err.Error())
		}
	}
}

func tempDatasetFile(t *testing.T, fileName string, ds *dataset.Dataset) (path string) {
	f, err := ioutil.TempFile("", fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(f).Encode(ds); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestDatasetRequestsForceSave(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	node := newTestQriNode(t)
	ref := addCitiesDataset(t, node)
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	_, err := inst.Dataset().Save(ctx, &SaveParams{Ref: ref.Alias()})
	if err == nil {
		t.Error("expected empty save without force flag to error")
	}

	_, err = inst.Dataset().Save(ctx, &SaveParams{
		Ref:   ref.Alias(),
		Force: true,
	})
	if err != nil {
		t.Errorf("expected empty save with force flag to not error. got: %q", err.Error())
	}
}

func TestDatasetRequestsSaveZip(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	// TODO (b5): import.zip has a ref.txt file that specifies test_user/test_repo as the dataset name,
	// save now requires a string reference. we need to pick a behaviour here & write a test that enforces it
	res, err := inst.Dataset().Save(ctx, &SaveParams{Ref: "me/huh", FilePaths: []string{"testdata/import.zip"}})
	if err != nil {
		t.Fatal(err.Error())
	}

	if res.Commit.Title != "Test Title" {
		t.Fatalf("Expected 'Test Title', got '%s'", res.Commit.Title)
	}
	if res.Meta.Title != "Test Repo" {
		t.Fatalf("Expected 'Test Repo', got '%s'", res.Meta.Title)
	}
}

func TestDatasetRequestsSaveApply(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	// Trying to save using apply without a transform is an error
	_, err := run.SaveWithParams(&SaveParams{
		Ref:      "me/cities_ds",
		BodyPath: "testdata/cities_2/body.csv",
		Apply:    true,
	})
	if err == nil {
		t.Fatal("expected an error, did not get one")
	}
	expectErr := `cannot apply while saving without a transform`
	if diff := cmp.Diff(expectErr, err.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):%s\n", diff)
	}

	// Save using apply and a transform, for a new dataset
	_, err = run.SaveWithParams(&SaveParams{
		Ref:       "me/hello",
		FilePaths: []string{"testdata/tf/transform.star"},
		Apply:     true,
	})
	if err != nil {
		t.Error(err)
	}

	// Save another dataset with a body
	_, err = run.SaveWithParams(&SaveParams{
		Ref:      "me/existing_ds",
		BodyPath: "testdata/cities_2/body.csv",
	})
	if err != nil {
		t.Error(err)
	}

	ds := run.MustGet(t, "me/existing_ds")
	bodyPath := ds.BodyPath

	// Save using apply and a transform, for dataset that already exists
	_, err = run.SaveWithParams(&SaveParams{
		Ref:       "me/existing_ds",
		FilePaths: []string{"testdata/cities_2/add_city.star"},
		Apply:     true,
	})
	if err != nil {
		t.Error(err)
	}

	ds = run.MustGet(t, "me/existing_ds")
	if ds.BodyPath == bodyPath {
		t.Error("expected body path to change, but it did not change")
	}

	// Save another dataset with a body
	_, err = run.SaveWithParams(&SaveParams{
		Ref:      "me/another_ds",
		BodyPath: "testdata/cities_2/body.csv",
	})
	if err != nil {
		t.Error(err)
	}

	ds = run.MustGet(t, "me/another_ds")
	bodyPath = ds.BodyPath

	// Save by adding a transform, but do not apply it. Body is unchanged.
	_, err = run.SaveWithParams(&SaveParams{
		Ref:       "me/another_ds",
		FilePaths: []string{"testdata/tf/transform.star"},
	})
	if err != nil {
		t.Error(err)
	}

	ds = run.MustGet(t, "me/another_ds")
	if ds.BodyPath != bodyPath {
		t.Error("unexpected: body path changed")
	}
}

func TestGet(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	ref, err := mr.GetRef(reporef.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Fatalf("error getting path: %s", err.Error())
	}

	moviesDs, err := dsfs.LoadDataset(ctx, mr.Filesystem(), ref.Path)
	if err != nil {
		t.Fatalf("error loading dataset: %s", err.Error())
	}

	moviesDs.OpenBodyFile(ctx, node.Repo.Filesystem())
	moviesBodyFile := moviesDs.BodyFile()
	reader, err := dsio.NewCSVReader(moviesDs.Structure, moviesBodyFile)
	if err != nil {
		t.Fatalf("creating CSV reader: %s", err)
	}
	moviesBody := mustBeArray(base.ReadEntries(reader))

	cases := []struct {
		description string
		params      *GetParams
		expect      interface{}
	}{

		{"empty ref",
			&GetParams{Ref: "", Selector: "body"}, `"" is not a valid dataset reference: empty reference`},

		{"invalid ref",
			&GetParams{Ref: "peer/ABC@abc"}, `"peer/ABC@abc" is not a valid dataset reference: unexpected character at position 8: '@'`},

		{"ref without path",
			&GetParams{Ref: "peer/movies"},
			setDatasetName(moviesDs, "peer/movies")},

		{"ref with path",
			&GetParams{Ref: fmt.Sprintf("peer/movies@%s", ref.Path)},
			setDatasetName(moviesDs, "peer/movies")},

		{"commit component",
			&GetParams{Ref: "peer/movies", Selector: "commit"},
			moviesDs.Commit},

		{"structure component",
			&GetParams{Ref: "peer/movies", Selector: "structure"},
			moviesDs.Structure},

		{"title field of commit component",
			&GetParams{Ref: "peer/movies", Selector: "commit.title"}, "initial commit"},

		{"body",
			&GetParams{Ref: "peer/movies", Selector: "body"}, moviesBody[:0]},

		{"body with limit and offfset",
			&GetParams{Ref: "peer/movies", Selector: "body",
				Limit: 5, Offset: 0, All: false}, moviesBody[:5]},

		{"body with invalid limit and offset",
			&GetParams{Ref: "peer/movies", Selector: "body",
				Limit: -5, Offset: -100, All: false}, "invalid limit / offset settings"},

		{"body with all flag ignores invalid limit and offset",
			&GetParams{Ref: "peer/movies", Selector: "body",
				Limit: -5, Offset: -100, All: true}, moviesBody},

		{"body with all flag",
			&GetParams{Ref: "peer/movies", Selector: "body",
				Limit: 0, Offset: 0, All: true}, moviesBody},

		{"body with limit and non-zero offset",
			&GetParams{Ref: "peer/movies", Selector: "body",
				Limit: 2, Offset: 10, All: false}, moviesBody[10:12]},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			got, err := inst.Dataset().Get(ctx, c.params)
			if err != nil {
				if err.Error() != c.expect {
					t.Errorf("error mismatch: expected: %s, got: %s", c.expect, err)
				}
				return
			}
			if ds, ok := got.Value.(*dataset.Dataset); ok {
				if ds.ID == "" {
					t.Errorf("returned dataset should have a non-empty ID field")
				}
			}
			if diff := cmp.Diff(c.expect, got.Value, cmpopts.IgnoreUnexported(
				dataset.Dataset{},
				dataset.Meta{},
				dataset.Commit{},
				dataset.Structure{},
				dataset.Viz{},
				dataset.Readme{},
				dataset.Transform{},
			),
				cmpopts.IgnoreFields(dataset.Dataset{}, "ID"),
			); diff != "" {
				t.Errorf("get output (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetParamsValidate(t *testing.T) {
	p := &GetParams{}
	p.Selector = "test+selector"
	expectErr := fmt.Errorf("could not parse request: invalid selector")
	if err := p.Validate(); err.Error() != expectErr.Error() {
		t.Errorf("GetParams.Validate error mismatch, expected %s, got %s", expectErr, err)
	}
}

func TestGetParamsSetNonZeroDefaults(t *testing.T) {
	gotParams := &GetParams{
		Selector: "body",
		Offset:   -1,
	}
	expectParams := &GetParams{
		Selector: "body",
		Limit:    25,
		Offset:   0,
	}
	gotParams.SetNonZeroDefaults()
	if diff := cmp.Diff(expectParams, gotParams); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

func TestGetParamsUnmarshalFromRequest(t *testing.T) {
	cases := []struct {
		description  string
		url          string
		expectParams *GetParams
		muxVars      map[string]string
	}{
		{
			"get request with ref",
			"/get/peer/my_ds",
			&GetParams{
				Ref: "peer/my_ds",
				All: true,
			},
			map[string]string{"ref": "peer/my_ds"},
		},
		{
			"get request with a selector",
			"/get/peer/my_ds/meta",
			&GetParams{
				Ref:      "peer/my_ds",
				Selector: "meta",
				All:      true,
			},
			map[string]string{"ref": "peer/my_ds", "selector": "meta"},
		},
		{
			"get request with limit and offset",
			"/get/peer/my_ds/body",
			&GetParams{
				Ref:      "peer/my_ds",
				Selector: "body",
				Limit:    0,
				Offset:   10,
			},
			map[string]string{"ref": "peer/my_ds", "selector": "body", "limit": "0", "offset": "10"},
		},
		{
			"get request with 'all'",
			"/get/peer/my_ds/body",
			&GetParams{
				Ref:      "peer/my_ds",
				Selector: "body",
				All:      true,
			},
			map[string]string{"ref": "peer/my_ds", "selector": "body", "all": "true"},
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			r := httptest.NewRequest("GET", c.url, nil)
			mustSetMuxVarsOnRequest(t, r, c.muxVars)
			gotParams := &GetParams{}
			err := gotParams.UnmarshalFromRequest(r)
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(c.expectParams, gotParams); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}

	badCases := []struct {
		description string
		url         string
		expectErr   string
		muxVars     map[string]string
	}{
		{
			"get me",
			"/get/me/my_ds",
			`username "me" not allowed`,
			map[string]string{"ref": "me/my_ds"},
		},
		{
			"bad parse",
			"/get/peer/my+ds",
			`unexpected character at position 7: '+'`,
			map[string]string{"ref": "peer/my+ds"},
		},
	}
	for i, c := range badCases {
		t.Run(c.description, func(t *testing.T) {
			r := httptest.NewRequest("GET", c.url, nil)
			mustSetMuxVarsOnRequest(t, r, c.muxVars)
			gotParams := &GetParams{}
			err := gotParams.UnmarshalFromRequest(r)
			if err == nil {
				t.Errorf("case %d: expected error, but did not get one", i)
				return
			}
			if diff := cmp.Diff(c.expectErr, err.Error()); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func mustSetMuxVarsOnRequest(t *testing.T, r *http.Request, muxVars map[string]string) {
	q := r.URL.Query()
	for varName, val := range muxVars {
		q.Add(varName, val)
	}
	r.URL.RawQuery = q.Encode()
	return
}

func TestGetZip(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	p := &GetParams{Ref: "peer/movies"}
	zipResults, err := inst.Dataset().GetZip(ctx, p)
	if err != nil {
		t.Fatalf("TestGetZip unexpected error: %s", err)
	}
	tempDir, err := ioutil.TempDir("", "get_zip_test")
	defer os.RemoveAll(tempDir)

	filename := path.Join(tempDir, "dataset.zip")
	if err := ioutil.WriteFile(filename, zipResults.Bytes, 0644); err != nil {
		t.Fatalf("error writing zip: %s", err)
	}
	expectedFiles := []string{
		"commit.json",
		"meta.json",
		"structure.json",
		"body.csv",
		"qri-ref.txt",
	}
	r, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatalf("error reading zip: %s", err)
	}
	gotFiles := []string{}
	for _, f := range r.File {
		gotFiles = append(gotFiles, f.Name)
	}

	if diff := cmp.Diff(expectedFiles, gotFiles); diff != "" {
		t.Errorf("expected zip files (-want +got):\n%s", diff)
	}
}

func TestGetCSV(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	ref, err := mr.GetRef(reporef.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Fatalf("error getting path: %s", err.Error())
	}
	moviesDs, err := dsfs.LoadDataset(ctx, mr.Filesystem(), ref.Path)
	if err != nil {
		t.Fatalf("error loading dataset: %s", err.Error())
	}
	moviesDs.OpenBodyFile(ctx, node.Repo.Filesystem())
	moviesBodyFile := moviesDs.BodyFile()
	expectedBytes, err := ioutil.ReadAll(moviesBodyFile)
	if err != nil {
		t.Fatalf("error reading body file: %s", err)
	}

	// the body file has `movie_title` for the first column, but the schema has the first column title as `title`
	expectedBytes = bytes.Replace(expectedBytes, []byte(`movie_title,duration`), []byte(`title,duration`), 1)

	gotBytes, err := inst.Dataset().GetCSV(ctx, &GetParams{Ref: "peer/movies", All: true})
	if err != nil {
		t.Fatalf("error getting csv: %s", err)
	}
	if diff := cmp.Diff(expectedBytes, gotBytes); diff != "" {
		t.Errorf("csv body bytes (-want +got):\n%s", diff)
	}
}

func TestGetBodySize(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	prevMaxBodySize := maxBodySizeToGetAll
	maxBodySizeToGetAll = 160
	defer func() {
		maxBodySizeToGetAll = prevMaxBodySize
	}()

	// Save a dataset with a body smaller than our test limit
	_, err := run.SaveWithParams(&SaveParams{
		Ref:      "me/small_ds",
		BodyPath: "testdata/cities_2/body.csv",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Save a dataset with a body larger than our test limit
	_, err = run.SaveWithParams(&SaveParams{
		Ref:      "me/large_ds",
		BodyPath: "testdata/cities_2/body_more.csv",
	})
	if err != nil {
		t.Fatal(err)
	}

	inst := run.Instance
	ctx := run.Ctx

	// Get the small dataset's body, which is okay
	params := GetParams{Ref: "me/small_ds", Selector: "body", Limit: -1, All: true}
	_, err = inst.Dataset().Get(ctx, &params)
	if err != nil {
		t.Errorf("%s", err)
	}

	// Get the large dataset's body, which will return an error
	params = GetParams{Ref: "me/large_ds", Selector: "body", Limit: -1, All: true}
	_, err = inst.Dataset().Get(ctx, &params)
	if err == nil {
		t.Errorf("expected error, did not get one")
	}
	expectErr := `body is too large to get all: 217 larger than 160`
	if err.Error() != expectErr {
		t.Errorf("error mismatch, expected: %s, got: %s", expectErr, err)
	}

	// Get the small dataset's body in CSV format, which is okay
	params = GetParams{Ref: "me/small_ds", Selector: "body", Limit: -1, All: true}
	_, err = inst.Dataset().GetCSV(ctx, &params)
	if err != nil {
		t.Errorf("%s", err)
	}

	// Get the large dataset's body in CSV format, which will return an error
	params = GetParams{Ref: "me/large_ds", Selector: "body", Limit: -1, All: true}
	_, err = inst.Dataset().GetCSV(ctx, &params)
	if err == nil {
		t.Errorf("expected error, did not get one")
	}
	if err.Error() != expectErr {
		t.Errorf("error mismatch, expected: %s, got: %s", expectErr, err)
	}
}

func setDatasetName(ds *dataset.Dataset, name string) *dataset.Dataset {
	parts := strings.Split(name, "/")
	ds.Peername = parts[0]
	ds.Name = parts[1]
	return ds
}

func componentToString(component interface{}, format string) string {
	switch format {
	case "json":
		bytes, err := json.MarshalIndent(component, "", " ")
		if err != nil {
			return err.Error()
		}
		return string(bytes)
	case "non-pretty json":
		bytes, err := json.Marshal(component)
		if err != nil {
			return err.Error()
		}
		return string(bytes)
	case "yaml":
		bytes, err := yaml.Marshal(component)
		if err != nil {
			return err.Error()
		}
		return string(bytes)
	default:
		return "Unknown format"
	}
}

func bodyToString(component interface{}) string {
	bytes, err := json.Marshal(component)
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

func bodyToPrettyString(component interface{}) string {
	bytes, err := json.MarshalIndent(component, "", " ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

func TestDatasetRequestsGetP2p(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	// Matches what is used to generated test peers.
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	factory := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 5)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	if err := p2ptest.ConnectNodes(ctx, testPeers); err != nil {
		t.Errorf("error connecting peers: %s", err.Error())
	}

	// Convert from test nodes to non-test nodes.
	peers := make([]*p2p.QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*p2p.QriNode)
	}

	var wg sync.WaitGroup
	for _, p1 := range peers {
		wg.Add(1)
		go func(node *p2p.QriNode) {
			defer wg.Done()
			// Get number from end of peername, use that to create dataset name.
			profile := node.Repo.Profiles().Owner(ctx)
			num := profile.Peername[len(profile.Peername)-1:]
			index, _ := strconv.ParseInt(num, 10, 32)
			name := datasets[index]
			ref := reporef.DatasetRef{Peername: profile.Peername, Name: name}

			inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
			// TODO (b5) - we're using "JSON" here b/c the "craigslist" test dataset
			// is tripping up the YAML serializer
			got, err := inst.Dataset().Get(ctx, &GetParams{Ref: fmt.Sprintf("%s/%s", profile.Peername, name)})
			if err != nil {
				t.Errorf("error getting dataset for %q: %s", ref, err.Error())
			}

			if got.Value == nil {
				t.Errorf("failed to get dataset for ref %q", ref)
			}
			// TODO: Test contents of Dataset.
		}(p1)
	}

	wg.Wait()
}

func TestDatasetRequestsRename(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	bad := []struct {
		p   *RenameParams
		err string
	}{
		{&RenameParams{}, "current name is required to rename a dataset"},
		{&RenameParams{Current: "peer/movies", Next: "peer/new movies"}, fmt.Sprintf("destination name: %s", dsref.ErrDescribeValidName.Error())},
		{&RenameParams{Current: "peer/cities", Next: "peer/sitemap"}, `dataset "peer/sitemap" already exists`},
	}

	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	for i, c := range bad {
		t.Run(fmt.Sprintf("bad_%d", i), func(t *testing.T) {
			_, err := inst.WithSource("local").Dataset().Rename(ctx, c.p)

			if err == nil {
				t.Fatalf("test didn't error")
			}

			if c.err != err.Error() {
				t.Errorf("error mismatch: expected: %s, got: %s", c.err, err)
			}
		})
	}

	log, err := mr.Logbook().DatasetRef(ctx, dsref.Ref{Username: "peer", Name: "movies"})
	if err != nil {
		t.Errorf("error getting logbook head reference: %s", err)
	}

	p := &RenameParams{
		Current: "peer/movies",
		Next:    "peer/new_movies",
	}

	res, err := inst.WithSource("local").Dataset().Rename(ctx, p)
	if err != nil {
		t.Errorf("unexpected error renaming: %s", err)
	}

	expect := &dsref.Ref{Username: "peer", Name: "new_movies"}
	if expect.Alias() != res.Alias() {
		t.Errorf("response mismatch. expected: %s, got: %s", expect.Alias(), res.Alias())
	}

	// get log by id this time
	after, err := mr.Logbook().Log(ctx, log.ID())
	if err != nil {
		t.Errorf("getting log by ID: %s", err)
	}

	if expect.Name != after.Name() {
		t.Errorf("rename log mismatch. expected: %s, got: %s", expect.Name, after.Name())
	}
}

func TestDatasetRequestsRemove(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	allRevs := &dsref.Rev{Field: "ds", Gen: -1}

	// create datasets working directory
	datasetsDir, err := ioutil.TempDir("", "QriTestDatasetRequestsRemove")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datasetsDir)

	// add a commit to craigslist
	_, err = inst.Dataset().Save(ctx, &SaveParams{Ref: "peer/craigslist", Dataset: &dataset.Dataset{Meta: &dataset.Meta{Title: "oh word"}}})
	if err != nil {
		t.Fatal(err)
	}

	badCases := []struct {
		err    string
		params RemoveParams
	}{
		{`"" is not a valid dataset reference: empty reference`, RemoveParams{Ref: "", Revision: allRevs}},
		{"reference not found", RemoveParams{Ref: "abc/not_found", Revision: allRevs}},
		{"can only remove whole dataset versions, not individual components", RemoveParams{Ref: "abc/not_found", Revision: &dsref.Rev{Field: "st", Gen: -1}}},
		{"invalid number of revisions to delete: 0", RemoveParams{Ref: "peer/movies", Revision: &dsref.Rev{Field: "ds", Gen: 0}}},
	}

	for i, c := range badCases {
		t.Run(fmt.Sprintf("bad_case_%s", c.err), func(t *testing.T) {
			_, err := inst.WithSource("local").Dataset().Remove(ctx, &c.params)

			if err == nil {
				t.Errorf("case %d: expected error. got nil", i)
				return
			} else if c.err != err.Error() {
				t.Errorf("case %d: error mismatch: expected: %s, got: %s", i, c.err, err)
			}
		})
	}

	goodCases := []struct {
		description string
		params      RemoveParams
		res         RemoveResponse
	}{
		{"all generations of peer/movies",
			RemoveParams{Ref: "peer/movies", Revision: allRevs},
			RemoveResponse{NumDeleted: -1},
		},
		{"all generations, specifying more revs than log length",
			RemoveParams{Ref: "peer/counter", Revision: &dsref.Rev{Field: "ds", Gen: 20}},
			RemoveResponse{NumDeleted: -1},
		},
	}

	for _, c := range goodCases {
		t.Run(fmt.Sprintf("good_case_%s", c.description), func(t *testing.T) {
			res, err := inst.WithSource("local").Dataset().Remove(ctx, &c.params)

			if err != nil {
				t.Errorf("unexpected error: %s", err)
				return
			}
			if c.res.NumDeleted != res.NumDeleted {
				t.Errorf("res.NumDeleted mismatch. want %d, got %d", c.res.NumDeleted, res.NumDeleted)
			}
			if c.res.Unlinked != res.Unlinked {
				t.Errorf("res.Unlinked mismatch. want %t, got %t", c.res.Unlinked, res.Unlinked)
			}
		})
	}
}

func TestDatasetRequestsPull(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	bad := []struct {
		p   PullParams
		err string
	}{
		{PullParams{Ref: "abc/hash###"}, "node is not online and no registry is configured"},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	for i, c := range bad {
		t.Run(fmt.Sprintf("bad_case_%d", i), func(t *testing.T) {
			_, err := inst.Dataset().Pull(ctx, &c.p)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if err.Error() == c.err {
				t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			}
		})
	}
}

func TestDatasetRequestsAddP2P(t *testing.T) {
	t.Skip("TODO (b5)")
	ctx, done := context.WithCancel(context.Background())
	defer done()

	// Matches what is used to generate the test peers.
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	// Create test nodes.
	factory := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 5)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	// Peers exchange Qri profile information.
	if err := p2ptest.ConnectNodes(ctx, testPeers); err != nil {
		t.Errorf("error upgrading to qri connections: %s", err.Error())
		return
	}

	// Convert from test nodes to non-test nodes.
	peers := make([]*p2p.QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*p2p.QriNode)
	}

	// Connect in memory Mapstore's behind the scene to simulate IPFS like behavior.
	for i, s0 := range peers {
		for _, s1 := range peers[i+1:] {
			m0 := (s0.Repo.Filesystem().Filesystem("mem")).(*qfs.MemFS)
			m1 := (s1.Repo.Filesystem().Filesystem("mem")).(*qfs.MemFS)
			m0.AddConnection(m1)
		}
	}

	var wg sync.WaitGroup
	for i, p0 := range peers {
		for _, p1 := range peers[i+1:] {
			wg.Add(1)
			go func(p0, p1 *p2p.QriNode) {
				defer wg.Done()

				// Get ref to dataset that peer2 has.
				profile := p1.Repo.Profiles().Owner(ctx)
				num := profile.Peername[len(profile.Peername)-1:]
				index, _ := strconv.ParseInt(num, 10, 32)
				name := datasets[index]
				ref := reporef.DatasetRef{Peername: profile.Peername, Name: name}
				p := &PullParams{
					Ref: ref.AliasString(),
				}

				// Build requests for peer1 to peer2.
				inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), p0)

				_, err := inst.Dataset().Pull(ctx, p)
				if err != nil {
					pro1 := p0.Repo.Profiles().Owner(ctx)
					pro2 := p1.Repo.Profiles().Owner(ctx)
					t.Errorf("error adding dataset for %s from %s to %s: %s",
						ref.Name, pro2.Peername, pro1.Peername, err.Error())
				}
			}(p0, p1)
		}
	}
	wg.Wait()

	// TODO: Validate that p1 has added data from p2.
}

func TestDatasetRequestsValidate(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	run := newTestRunner(t)
	defer run.Delete()

	movieb := `Avatar ,178
Pirates of the Caribbean: At World's End ,169
Pirates of the Caribbean: At World's End ,foo
`
	schemaB := `{
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
	}`

	bodyFilename := run.MakeTmpFilename("data.csv")
	schemaFilename := run.MakeTmpFilename("schema.json")
	run.MustWriteFile(t, bodyFilename, movieb)
	run.MustWriteFile(t, schemaFilename, schemaB)

	cases := []struct {
		p         ValidateParams
		numErrors int
		err       string
		isNil     bool
	}{
		{ValidateParams{Ref: ""}, 0, "bad arguments provided", true},
		{ValidateParams{Ref: "me"}, 0, "\"me\" is not a valid dataset reference: need username separated by '/' from dataset name", true},
		{ValidateParams{Ref: "me/movies"}, 4, "", false},
		{ValidateParams{Ref: "me/movies", BodyFilename: bodyFilename}, 1, "", false},
		{ValidateParams{Ref: "me/movies", SchemaFilename: schemaFilename}, 5, "", false},
		{ValidateParams{SchemaFilename: schemaFilename, BodyFilename: bodyFilename}, 1, "", false},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
	for i, c := range cases {
		res, err := inst.WithSource("local").Dataset().Validate(ctx, &c.p)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err.Error())
			continue
		}

		if res == nil && !c.isNil {
			t.Errorf("case %d error result was nil: expected result to not be nil", i)
			continue
		}

		if res != nil && len(res.Errors) != c.numErrors {
			t.Errorf("case %d error count mismatch. expected: %d, got: %d", i, c.numErrors, len(res.Errors))
			continue
		}
	}
}

func TestDatasetRequestsStats(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	badCases := []struct {
		description string
		ref         string
		expectedErr string
	}{
		{"empty reference", "", `"" is not a valid dataset reference: empty reference`},
		{"bad reference", "!", `"!" is not a valid dataset reference: unexpected character at position 0: '!'`},
		{"dataset does not exist", "me/dataset_does_not_exist", "reference not found"},
	}
	for _, c := range badCases {
		t.Run(c.description, func(t *testing.T) {
			_, err = inst.WithSource("local").Dataset().Get(ctx, &GetParams{Ref: c.ref, Selector: "stats"})
			if c.expectedErr != err.Error() {
				t.Errorf("error mismatch, expected: %q, got: %q", c.expectedErr, err.Error())
			}
		})
	}

	// TODO (ramfox): see if there is a better way to verify the stat bytes then
	// just inputing them in the cases struct
	goodCases := []struct {
		description string
		ref         string
		expectPath  string
	}{
		{"csv: me/cities", "me/cities", "./testdata/cities.stats.json"},
		{"json: me/sitemap", "me/sitemap", `./testdata/sitemap.stats.json`},
	}
	for _, c := range goodCases {
		t.Run(c.description, func(t *testing.T) {
			res, err := inst.WithSource("local").Dataset().Get(ctx, &GetParams{Ref: c.ref, Selector: "stats"})
			if err != nil {
				t.Fatalf("unexpected error: %q", err.Error())
			}
			expectData, err := ioutil.ReadFile(c.expectPath)
			if err != nil {
				t.Fatal(err)
			}

			expect := []interface{}{}
			if err = json.Unmarshal(expectData, &expect); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expect, res.Value); diff != "" {
				t.Errorf("result mismatch (-want +got):%s\n", diff)
				output, _ := json.Marshal(res.Value)
				fmt.Println(string(output))
			}
		})
	}
}

// Convert the interface value into an array, or panic if not possible
func mustBeArray(i interface{}, err error) []interface{} {
	if err != nil {
		panic(err)
	}
	return i.([]interface{})
}

func TestFormFileDataset(t *testing.T) {
	r := newFormFileRequest(t, nil, nil)
	dsp := &dataset.Dataset{}
	if err := formFileDataset(r, dsp); err != nil {
		t.Error("expected 'empty' request to be ok")
	}

	r = newFormFileRequest(t, map[string]string{
		"file":      dstestTestdataFile("complete/input.dataset.json"),
		"viz":       dstestTestdataFile("complete/template.html"),
		"transform": dstestTestdataFile("complete/transform.star"),
		"body":      dstestTestdataFile("complete/body.csv"),
	}, nil)
	if err := formFileDataset(r, dsp); err != nil {
		t.Error(err)
	}

	r = newFormFileRequest(t, map[string]string{
		"file": "testdata/dataset.yml",
		"body": dstestTestdataFile("complete/body.csv"),
	}, nil)
	if err := formFileDataset(r, dsp); err != nil {
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

func dstestTestdataFile(path string) string {
	_, currfile, _, _ := runtime.Caller(0)
	testdataPath := filepath.Join(filepath.Dir(currfile), "testdata")
	return filepath.Join(testdataPath, path)
}
