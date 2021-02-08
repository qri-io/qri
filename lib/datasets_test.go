package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
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
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
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

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)

	privateErrMsg := "option to make dataset private not yet implemented, refer to https://github.com/qri-io/qri/issues/291 for updates"
	_, err = m.Save(ctx, &SaveParams{Private: true})
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
		{"meta set title", SaveParams{Ref: "me/cities", FilePaths: []string{citiesMetaOnePath}}, nil},
		{"meta set description, supply same body", SaveParams{Ref: "me/cities", FilePaths: []string{citiesMetaTwoPath}, BodyPath: s.URL + "/body.csv"}, nil},
	}

	for i, c := range good {
		got, err := m.Save(ctx, &c.params)
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
		_, err := m.Save(ctx, &c.params)
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
	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)

	_, err := m.Save(ctx, &SaveParams{Ref: ref.Alias()})
	if err == nil {
		t.Error("expected empty save without force flag to error")
	}

	_, err = m.Save(ctx, &SaveParams{
		Ref:   ref.Alias(),
		Force: true,
	})
	if err != nil {
		t.Errorf("expected empty save with flag to not error. got: %s", err.Error())
	}
}

func TestDatasetRequestsSaveZip(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)

	// TODO (b5): import.zip has a ref.txt file that specifies test_user/test_repo as the dataset name,
	// save now requires a string reference. we need to pick a behaviour here & write a test that enforces it
	res, err := m.Save(ctx, &SaveParams{Ref: "me/huh", FilePaths: []string{"testdata/import.zip"}})
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

func TestDatasetRequestsList(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	var (
		movies, counter, cities, craigslist, sitemap dsref.VersionInfo
	)

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err)
		return
	}

	refs, err := mr.References(0, 30)
	if err != nil {
		t.Fatalf("error getting namespace: %s", err)
	}

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err)
	}

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)

	for _, ref := range refs {
		dr := reporef.ConvertToVersionInfo(&ref)
		switch dr.Name {
		case "movies":
			movies = dr
		case "counter":
			counter = dr
		case "cities":
			cities = dr
		case "craigslist":
			craigslist = dr
		case "sitemap":
			sitemap = dr
		}
	}

	cases := []struct {
		description string
		p           *ListParams
		res         []dsref.VersionInfo
		err         string
	}{
		{"list datasets - empty (default)", &ListParams{}, []dsref.VersionInfo{cities, counter, craigslist, movies, sitemap}, ""},
		{"list datasets - weird (returns sensible default)", &ListParams{OrderBy: "chaos", Limit: -33, Offset: -50}, []dsref.VersionInfo{cities, counter, craigslist, movies, sitemap}, ""},
		{"list datasets - happy path", &ListParams{OrderBy: "", Limit: 30, Offset: 0}, []dsref.VersionInfo{cities, counter, craigslist, movies, sitemap}, ""},
		{"list datasets - limit 2 offset 0", &ListParams{OrderBy: "", Limit: 2, Offset: 0}, []dsref.VersionInfo{cities, counter}, ""},
		{"list datasets - limit 2 offset 2", &ListParams{OrderBy: "", Limit: 2, Offset: 2}, []dsref.VersionInfo{craigslist, movies}, ""},
		{"list datasets - limit 2 offset 4", &ListParams{OrderBy: "", Limit: 2, Offset: 4}, []dsref.VersionInfo{sitemap}, ""},
		{"list datasets - limit 2 offset 5", &ListParams{OrderBy: "", Limit: 2, Offset: 5}, []dsref.VersionInfo{}, ""},
		{"list datasets - order by timestamp", &ListParams{OrderBy: "timestamp", Limit: 30, Offset: 0}, []dsref.VersionInfo{cities, counter, craigslist, movies, sitemap}, ""},
		{"list datasets - peername 'me'", &ListParams{Peername: "me", OrderBy: "timestamp", Limit: 30, Offset: 0}, []dsref.VersionInfo{cities, counter, craigslist, movies, sitemap}, ""},
		// TODO: re-enable {&ListParams{OrderBy: "name", Limit: 30, Offset: 0}, []*dsref.VersionInfo{cities, counter, movies}, ""},
	}

	m := NewDatasetMethods(inst)
	for _, c := range cases {
		got, err := m.List(ctx, c.p)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case '%s' error mismatch: expected: %s, got: %s", c.description, c.err, err)
			continue
		}

		if c.err == "" && c.res != nil {
			if len(c.res) != len(got) {
				t.Errorf("case '%s' response length mismatch. expected %d, got: %d", c.description, len(c.res), len(got))
				continue
			}

			for j, expect := range c.res {
				if err := compareVersionInfoAsSimple(expect, got[j]); err != nil {
					t.Errorf("case '%s' expected dataset error. index %d mismatch: %s", c.description, j, err.Error())
					continue
				}
			}
		}
	}
}

func compareVersionInfoAsSimple(a, b dsref.VersionInfo) error {
	if a.ProfileID != b.ProfileID {
		return fmt.Errorf("PeerID mismatch. %s != %s", a.ProfileID, b.ProfileID)
	}
	if a.Username != b.Username {
		return fmt.Errorf("Peername mismatch. %s != %s", a.Username, b.Username)
	}
	if a.Name != b.Name {
		return fmt.Errorf("Name mismatch. %s != %s", a.Name, b.Name)
	}
	if a.Path != b.Path {
		return fmt.Errorf("Path mismatch. %s != %s", a.Path, b.Path)
	}
	return nil
}

func TestDatasetRequestsListP2p(t *testing.T) {
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

			inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
			m := NewDatasetMethods(inst)
			p := &ListParams{OrderBy: "", Limit: 30, Offset: 0}
			res, err := m.List(ctx, p)
			if err != nil {
				t.Errorf("error listing dataset: %s", err.Error())
			}
			// Get number from end of peername, use that to find dataset name.
			profile, _ := node.Repo.Profile(ctx)
			num := profile.Peername[len(profile.Peername)-1:]
			index, _ := strconv.ParseInt(num, 10, 32)
			expect := datasets[index]

			if res[0].Name != expect {
				t.Errorf("dataset %s mismatch: %s", res[0].Name, expect)
			}
		}(p1)
	}

	wg.Wait()
}

func TestDatasetRequestsGet(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)

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

	prettyJSONConfig, _ := dataset.NewJSONOptions(map[string]interface{}{"pretty": true})
	nonprettyJSONConfig, _ := dataset.NewJSONOptions(map[string]interface{}{"pretty": false})

	cases := []struct {
		description string
		params      *GetParams
		expect      string
	}{
		{"invalid peer name",
			&GetParams{Refstr: "peer/ABC@abc"}, `"peer/ABC@abc" is not a valid dataset reference: unexpected character at position 8: '@'`},

		{"peername without path",
			&GetParams{Refstr: "peer/movies"},
			componentToString(setDatasetName(moviesDs, "peer/movies"), "yaml")},

		{"peername with path",
			&GetParams{Refstr: fmt.Sprintf("peer/movies@%s", ref.Path)},
			componentToString(setDatasetName(moviesDs, "peer/movies"), "yaml")},

		{"peername as json format",
			&GetParams{Refstr: "peer/movies", Format: "json"},
			componentToString(setDatasetName(moviesDs, "peer/movies"), "json")},

		{"commit component",
			&GetParams{Refstr: "peer/movies", Selector: "commit"},
			componentToString(moviesDs.Commit, "yaml")},

		{"commit component as json format",
			&GetParams{Refstr: "peer/movies", Selector: "commit", Format: "json"},
			componentToString(moviesDs.Commit, "json")},

		{"title field of commit component",
			&GetParams{Refstr: "peer/movies", Selector: "commit.title"}, "initial commit\n"},

		{"title field of commit component as json",
			&GetParams{Refstr: "peer/movies", Selector: "commit.title", Format: "json"},
			"\"initial commit\""},

		{"title field of commit component as yaml",
			&GetParams{Refstr: "peer/movies", Selector: "commit.title", Format: "yaml"},
			"initial commit\n"},

		{"title field of commit component as mispelled format",
			&GetParams{Refstr: "peer/movies", Selector: "commit.title", Format: "jason"},
			"unknown format: \"jason\""},

		{"body as json",
			&GetParams{Refstr: "peer/movies", Selector: "body", Format: "json"}, "[]"},

		{"dataset empty",
			&GetParams{Refstr: "", Selector: "body", Format: "json"}, `"" is not a valid dataset reference: empty reference`},

		{"body as csv",
			&GetParams{Refstr: "peer/movies", Selector: "body", Format: "csv"}, "title,duration\n"},

		{"body with limit and offfset",
			&GetParams{Refstr: "peer/movies", Selector: "body", Format: "json",
				Limit: 5, Offset: 0, All: false}, bodyToString(moviesBody[:5])},

		{"body with invalid limit and offset",
			&GetParams{Refstr: "peer/movies", Selector: "body", Format: "json",
				Limit: -5, Offset: -100, All: false}, "invalid limit / offset settings"},

		{"body with all flag ignores invalid limit and offset",
			&GetParams{Refstr: "peer/movies", Selector: "body", Format: "json",
				Limit: -5, Offset: -100, All: true}, bodyToString(moviesBody)},

		{"body with all flag",
			&GetParams{Refstr: "peer/movies", Selector: "body", Format: "json",
				Limit: 0, Offset: 0, All: true}, bodyToString(moviesBody)},

		{"body with limit and non-zero offset",
			&GetParams{Refstr: "peer/movies", Selector: "body", Format: "json",
				Limit: 2, Offset: 10, All: false}, bodyToString(moviesBody[10:12])},

		{"head non-pretty json",
			&GetParams{Refstr: "peer/movies", Format: "json", FormatConfig: nonprettyJSONConfig},
			componentToString(setDatasetName(moviesDs, "peer/movies"), "non-pretty json")},

		{"body pretty json",
			&GetParams{Refstr: "peer/movies", Selector: "body", Format: "json",
				FormatConfig: prettyJSONConfig, Limit: 3, Offset: 0, All: false},
			bodyToPrettyString(moviesBody[:3])},
	}

	dsm := NewDatasetMethods(inst)
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			got, err := dsm.Get(ctx, c.params)
			if err != nil {
				if err.Error() != c.expect {
					t.Errorf("error mismatch: expected: %s, got: %s", c.expect, err)
				}
				return
			}

			result := string(got.Bytes)
			if result != c.expect {
				t.Errorf("result mismatch expected:\n%q, got:\n%q", c.expect, result)
			}
		})
	}
}

func TestDatasetRequestsGetFSIPath(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)

	tempDir, err := ioutil.TempDir("", "get_fsi_test")
	defer os.RemoveAll(tempDir)

	dsDir := filepath.Join(tempDir, "movies")
	fsim := NewFSIMethods(inst)
	p := &CheckoutParams{
		Dir: dsDir,
		Ref: "peer/movies",
	}
	out := ""
	if err := fsim.Checkout(p, &out); err != nil {
		t.Fatalf("error checking out dataset: %s", err)
	}

	dsm := NewDatasetMethods(inst)
	getParams := &GetParams{
		Refstr: "peer/movies",
	}
	got, err := dsm.Get(ctx, getParams)
	if err != nil {
		t.Fatalf("error getting fsi dataset: %s", err)
	}
	if got.Ref.Username != "peer" {
		t.Errorf("incorrect Username, expected 'peer', got %q", got.Ref.Username)
	}
	if got.Ref.Name != "movies" {
		t.Errorf("incorrect Username, expected 'movies', got %q", got.Ref.Name)
	}
	if got.FSIPath != dsDir {
		t.Errorf("incorrect FSIPath, expected %q, got %q", dsDir, got.FSIPath)
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
			profile, _ := node.Repo.Profile(ctx)
			num := profile.Peername[len(profile.Peername)-1:]
			index, _ := strconv.ParseInt(num, 10, 32)
			name := datasets[index]
			ref := reporef.DatasetRef{Peername: profile.Peername, Name: name}

			inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
			m := NewDatasetMethods(inst)
			// TODO (b5) - we're using "JSON" here b/c the "craigslist" test dataset
			// is tripping up the YAML serializer
			got, err := m.Get(ctx, &GetParams{Refstr: ref.String(), Format: "json"})
			if err != nil {
				t.Errorf("error getting dataset for %q: %s", ref, err.Error())
			}

			if got.Bytes == nil {
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
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
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

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)
	for i, c := range bad {
		t.Run(fmt.Sprintf("bad_%d", i), func(t *testing.T) {
			got := &dsref.VersionInfo{}
			err := m.Rename(c.p, got)

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

	res := &dsref.VersionInfo{}
	if err := m.Rename(p, res); err != nil {
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

func TestRenameNoHistory(t *testing.T) {
	ctx := context.Background()
	tr := newTestRunner(t)
	defer tr.Delete()

	workDir := tr.CreateAndChdirToWorkDir("remove_no_history")
	initP := &InitDatasetParams{
		TargetDir: workDir,
		Name:      "remove_no_history",
		Format:    "csv",
	}
	var refstr string
	if err := NewFSIMethods(tr.Instance).InitDataset(ctx, initP, &refstr); err != nil {
		t.Fatal(err)
	}

	// 	// Read .qri-ref file, it contains the reference this directory is linked to
	actual := tr.MustReadFile(t, filepath.Join(workDir, ".qri-ref"))
	expect := "peer/remove_no_history"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("qri list (-want +got):\n%s", diff)
	}

	// Rename before any commits have happened
	renameP := &RenameParams{
		Current: "me/remove_no_history",
		Next:    "me/remove_second_name",
	}
	res := new(dsref.VersionInfo)
	if err := NewDatasetMethods(tr.Instance).Rename(renameP, res); err != nil {
		t.Fatal(err)
	}

	// Read .qri-ref file, it contains the new reference name
	actual = tr.MustReadFile(t, filepath.Join(workDir, ".qri-ref"))
	expect = "peer/remove_second_name"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("qri list (-want +got):\n%s", diff)
	}
}

func TestDatasetRequestsRemove(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	dsm := NewDatasetMethods(inst)
	allRevs := dsref.Rev{Field: "ds", Gen: -1}

	// we need some fsi stuff to fully test remove
	fsim := NewFSIMethods(inst)
	// create datasets working directory
	datasetsDir, err := ioutil.TempDir("", "QriTestDatasetRequestsRemove")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datasetsDir)

	// initialize an example no-history dataset
	initp := &InitDatasetParams{
		Name:      "no_history",
		TargetDir: filepath.Join(datasetsDir, "no_history"),
		Format:    "csv",
	}
	var noHistoryName string
	if err := fsim.InitDataset(ctx, initp, &noHistoryName); err != nil {
		t.Fatal(err)
	}

	// link cities dataset with a checkout
	checkoutp := &CheckoutParams{
		Dir: filepath.Join(datasetsDir, "cities"),
		Ref: "me/cities",
	}
	var out string
	if err := fsim.Checkout(checkoutp, &out); err != nil {
		t.Fatal(err)
	}

	// add a commit to craigslist
	_, err = dsm.Save(ctx, &SaveParams{Ref: "peer/craigslist", Dataset: &dataset.Dataset{Meta: &dataset.Meta{Title: "oh word"}}})
	if err != nil {
		t.Fatal(err)
	}

	// link craigslist with a checkout
	checkoutp = &CheckoutParams{
		Dir: filepath.Join(datasetsDir, "craigslist"),
		Ref: "me/craigslist",
	}
	if err := fsim.Checkout(checkoutp, &out); err != nil {
		t.Fatal(err)
	}

	badCases := []struct {
		err    string
		params RemoveParams
	}{
		{"repo: empty dataset reference", RemoveParams{Ref: "", Revision: allRevs}},
		{"repo: not found", RemoveParams{Ref: "abc/ABC", Revision: allRevs}},
		{"can only remove whole dataset versions, not individual components", RemoveParams{Ref: "abc/ABC", Revision: dsref.Rev{Field: "st", Gen: -1}}},
		{"invalid number of revisions to delete: 0", RemoveParams{Ref: "peer/movies", Revision: dsref.Rev{Field: "ds", Gen: 0}}},
		{"dataset is not linked to filesystem, cannot use keep-files", RemoveParams{Ref: "peer/movies", Revision: allRevs, KeepFiles: true}},
	}

	for i, c := range badCases {
		t.Run(fmt.Sprintf("bad_case_%s", c.err), func(t *testing.T) {
			res := RemoveResponse{}
			err := dsm.Remove(&c.params, &res)

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
			RemoveParams{Ref: "peer/counter", Revision: dsref.Rev{Field: "ds", Gen: 20}},
			RemoveResponse{NumDeleted: -1},
		},
	}

	for _, c := range goodCases {
		t.Run(fmt.Sprintf("good_case_%s", c.description), func(t *testing.T) {
			res := RemoveResponse{}
			err := dsm.Remove(&c.params, &res)

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
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)
	for i, c := range bad {
		t.Run(fmt.Sprintf("bad_case_%d", i), func(t *testing.T) {
			got := &dataset.Dataset{}
			err := m.Pull(&c.p, got)
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
				profile, _ := p1.Repo.Profile(ctx)
				num := profile.Peername[len(profile.Peername)-1:]
				index, _ := strconv.ParseInt(num, 10, 32)
				name := datasets[index]
				ref := reporef.DatasetRef{Peername: profile.Peername, Name: name}
				p := &PullParams{
					Ref: ref.AliasString(),
				}

				// Build requests for peer1 to peer2.
				inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), p0)
				dsm := NewDatasetMethods(inst)
				got := &dataset.Dataset{}

				err := dsm.Pull(p, got)
				if err != nil {
					pro1, _ := p0.Repo.Profile(ctx)
					pro2, _ := p1.Repo.Profile(ctx)
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
	}{
		{ValidateParams{Ref: ""}, 0, "bad arguments provided"},
		{ValidateParams{Ref: "me"}, 0, "\"me\" is not a valid dataset reference: need username separated by '/' from dataset name"},
		{ValidateParams{Ref: "me/movies"}, 4, ""},
		{ValidateParams{Ref: "me/movies", BodyFilename: bodyFilename}, 1, ""},
		{ValidateParams{Ref: "me/movies", SchemaFilename: schemaFilename}, 5, ""},
		{ValidateParams{SchemaFilename: schemaFilename, BodyFilename: bodyFilename}, 1, ""},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)
	for i, c := range cases {
		res := &ValidateResponse{}
		err := m.Validate(&c.p, res)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err.Error())
			continue
		}

		if len(res.Errors) != c.numErrors {
			t.Errorf("case %d error count mismatch. expected: %d, got: %d", i, c.numErrors, len(res.Errors))
			continue
		}
	}
}

func TestDatasetRequestsValidateFSI(t *testing.T) {
	ctx := context.Background()
	tr := newTestRunner(t)
	defer tr.Delete()

	workDir := tr.CreateAndChdirToWorkDir("remove_no_history")
	initP := &InitDatasetParams{
		Name:      "validate_test",
		TargetDir: filepath.Join(workDir, "validate_test"),
		Format:    "csv",
	}
	var refstr string
	if err := NewFSIMethods(tr.Instance).InitDataset(ctx, initP, &refstr); err != nil {
		t.Fatal(err)
	}

	m := NewDatasetMethods(tr.Instance)

	vp := &ValidateParams{Ref: refstr}
	vr := &ValidateResponse{}
	if err := m.Validate(vp, vr); err != nil {
		t.Fatal(err)
	}
}

func TestDatasetRequestsStats(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)

	badCases := []struct {
		description string
		ref         string
		expectedErr string
	}{
		{"empty reference", "", `either a reference or dataset is required`},
		{"empty reference", "///", `"///" is not a valid dataset reference: unexpected character at position 0: '/'`},
		{"dataset does not exist", "me/dataset_does_not_exist", "reference not found"},
	}
	for i, c := range badCases {
		res := &dataset.Stats{}
		err := m.Stats(&StatsParams{Ref: c.ref}, res)
		if c.expectedErr != err.Error() {
			t.Errorf("%d. case %s: error mismatch, expected: '%s', got: '%s'", i, c.description, c.expectedErr, err.Error())
		}
	}

	// TODO (ramfox): see if there is a better way to verify the stat bytes then
	// just inputing them in the cases struct
	goodCases := []struct {
		description string
		ref         string
		expected    []byte
	}{
		{"csv: me/cities", "me/cities", []byte(`[{"count":5,"frequencies":{"chatham":1,"chicago":1,"new york":1,"raleigh":1,"toronto":1},"maxLength":8,"minLength":7,"type":"string","unique":5},{"count":5,"histogram":{"bins":[35000,250000,300000,8500000,40000000,40000001],"frequencies":[1,1,1,1,1]},"max":40000000,"mean":9817000,"median":300000,"min":35000,"type":"numeric"},{"count":5,"histogram":{"bins":[44.4,50.65,55.5,65.25,66.25],"frequencies":[2,1,1,1]},"max":65.25,"mean":52.04,"median":50.65,"min":44.4,"type":"numeric"},{"count":5,"falseCount":1,"trueCount":4,"type":"boolean"}]`)},
		{"json: me/sitemap", "me/sitemap", []byte(`[{"count":10,"histogram":{"bins":[24515,24552,25028,25329,27827,28291,28337,30258,34042,40079,40080],"frequencies":[1,1,1,1,1,1,1,1,1,1]},"key":"contentLength","max":40079,"mean":28825.8,"median":28291,"min":24515,"type":"numeric"},{"count":10,"frequencies":{"text/html; charset=utf-8":10},"key":"contentSniff","maxLength":24,"minLength":24,"type":"string","unique":1},{"count":10,"frequencies":{"text/html; charset=utf-8":10},"key":"contentType","maxLength":24,"minLength":24,"type":"string","unique":1},{"count":10,"histogram":{"bins":[74291866,89911449,4055332831,4075146079,4077173686,4077286486,4077931202,4080164896,4080183198,4081577841,4081577842],"frequencies":[1,1,1,1,1,1,1,1,1,1]},"key":"duration","max":4081577841,"mean":3276899953.4,"median":4077286486,"min":74291866,"type":"numeric"},{"count":10,"frequencies":{"12200c610d5ec64231b2751e8ede38b4fd7d911360159fc5bba3e165f68c1ee4f169":1,"1220131c6c4233a75f1361045dcb45173c4d13eb94f08184edb45109975ce7d8b33a":1,"12203236442fb7b71bf1696b6071beb4abcc08bf318ade377daf352fdc846f14a292":1,"12203f978c899c51c0ee60a2f983ed76d2cd9351846e98efeb8f2f1d025e2e39dff8":1,"122055b62b100b92467d64d781e7f14a91e7ffac0869cb7ecc7fc38ad620d8c04ef5":1,"122066ace8ef380db026ef249b1a4cd2b35008c8901ab98994c56ab8022f099e7991":1,"12206c2e9e217a9efaa0506eba68a039513cd2ec8c7025c367b9b64256d462fb1660":1,"122075ddcf3989ef97e5f9d59ceb1b5b71d72bc32d8b76c181d7826b028792e202ab":1,"122093f53b43ac1e56bd091abacd6c1813eb249400ed0c98eba4e667916a4286ccf2":1,"1220e1975125bdbc7638bb16bd1a52e2115b6531511def6a0228ad1b671111a8066f":1},"key":"hash","maxLength":68,"minLength":68,"type":"string","unique":10},{"key":"links","type":"array","values":[{"count":10,"frequencies":{"http://cfpub.epa.gov/locator":1,"http://epa.gov/ace":1,"http://epa.gov/ace/ace-biomonitoring-lead":1,"http://epa.gov/careers":1,"http://epa.gov/environmental-topics":1,"http://epa.gov/environmental-topics/greener-living":1,"http://epa.gov/home/grants-and-other-funding-opportunities":1,"http://epa.gov/lead":1,"http://epa.gov/open":1,"http://usa.gov":1},"maxLength":58,"minLength":14,"unique":10},{"count":10,"frequencies":{"http://epa.gov/ace/ace-biomonitoring-mercury":1,"http://epa.gov/ace/ace-update-history":1,"http://epa.gov/contracts":1,"http://epa.gov/environmental-topics/air-topics":1,"http://epa.gov/environmental-topics/health-topics":1,"http://epa.gov/mold":1,"http://epa.gov/ocr/whistleblower-protections-epa-and-how-they-relate-non-disclosure-agreements-signed-epa-employees":1,"http://epa.gov/planandbudget":1,"http://regulations.gov":1,"http://whitehouse.gov":1},"maxLength":115,"minLength":19,"unique":10},{"count":10,"frequencies":{"http://epa.gov/ace/ace-biomonitoring-cotinine":1,"http://epa.gov/ace/americas-children-and-environment-update-listserv":1,"http://epa.gov/bedbugs":1,"http://epa.gov/careers":1,"http://epa.gov/environmental-topics/land-waste-and-cleanup-topics":1,"http://epa.gov/home/forms/contact-epa":1,"http://epa.gov/home/grants-and-other-funding-opportunities":1,"http://epa.gov/newsroom/email-subscriptions":1,"http://epa.gov/pesticides":1,"http://epa.gov/privacy":1},"maxLength":68,"minLength":22,"unique":10},{"count":10,"frequencies":{"http://epa.gov/ace/ace-biomonitoring-perfluorochemicals-pfcs":1,"http://epa.gov/ace/basic-information-about-ace":1,"http://epa.gov/contracts":1,"http://epa.gov/environmental-topics/chemicals-and-toxics-topics":1,"http://epa.gov/home/epa-hotlines":1,"http://epa.gov/lead":1,"http://epa.gov/ocr/whistleblower-protections-epa-and-how-they-relate-non-disclosure-agreements-signed-epa-employees":1,"http://epa.gov/privacy/privacy-and-security-notice":1,"http://epa.gov/radon":1,"http://usa.gov":1},"maxLength":115,"minLength":14,"unique":10},{"count":9,"frequencies":{"http://data.gov":1,"http://epa.gov/ace/ace-biomonitoring-polychlorinated-biphenyls-pcbs":1,"http://epa.gov/ace/key-findings-ace3-report":1,"http://epa.gov/environmental-topics/environmental-information-location":1,"http://epa.gov/environmental-topics/science-topics":1,"http://epa.gov/foia":1,"http://epa.gov/home/grants-and-other-funding-opportunities":1,"http://epa.gov/privacy":1,"http://whitehouse.gov":1},"maxLength":70,"minLength":15,"unique":9},{"count":9,"frequencies":{"http://epa.gov/ace/ace-biomonitoring-polybrominated-diphenyl-ethers-pbdes":1,"http://epa.gov/ace/ace-frequent-questions":1,"http://epa.gov/environmental-topics/greener-living":1,"http://epa.gov/environmental-topics/water-topics":1,"http://epa.gov/home/forms/contact-epa":1,"http://epa.gov/home/frequent-questions-specific-epa-programstopics":1,"http://epa.gov/ocr/whistleblower-protections-epa-and-how-they-relate-non-disclosure-agreements-signed-epa-employees":1,"http://epa.gov/office-inspector-general/about-epas-office-inspector-general":1,"http://epa.gov/privacy/privacy-and-security-notice":1},"maxLength":115,"minLength":37,"unique":9},{"count":9,"frequencies":{"http://data.gov":1,"http://epa.gov/ace/ace-biomonitoring-phthalates":1,"http://epa.gov/ace/ace-environments-and-contaminants":1,"http://epa.gov/environmental-topics/health-topics":1,"http://epa.gov/environmental-topics/z-index":1,"http://epa.gov/home/epa-hotlines":1,"http://epa.gov/newsroom":1,"http://epa.gov/privacy":1,"http://facebook.com/EPA":1},"maxLength":52,"minLength":15,"unique":9},{"count":9,"frequencies":{"http://epa.gov/ace/ace-biomonitoring":1,"http://epa.gov/ace/ace-biomonitoring-bisphenol-bpa":1,"http://epa.gov/environmental-topics/land-waste-and-cleanup-topics":1,"http://epa.gov/foia":1,"http://epa.gov/laws-regulations":1,"http://epa.gov/office-inspector-general/about-epas-office-inspector-general":1,"http://epa.gov/open":1,"http://epa.gov/privacy/privacy-and-security-notice":1,"http://twitter.com/epa":1},"maxLength":75,"minLength":19,"unique":9},{"count":9,"frequencies":{"http://data.gov":1,"http://epa.gov/ace/ace-biomonitoring-perchlorate":1,"http://epa.gov/ace/ace-health":1,"http://epa.gov/home/frequent-questions-specific-epa-programstopics":1,"http://epa.gov/lead":1,"http://epa.gov/newsroom":1,"http://epa.gov/regulatory-information-sector":1,"http://regulations.gov":1,"http://youtube.com/user/USEPAgov":1},"maxLength":66,"minLength":15,"unique":9},{"count":7,"frequencies":{"http://epa.gov/ace/ace-supplementary-topics":1,"http://epa.gov/mold":1,"http://epa.gov/office-inspector-general/about-epas-office-inspector-general":1,"http://epa.gov/open":1,"http://epa.gov/regulatory-information-topic":1,"http://facebook.com/EPA":1,"http://flickr.com/photos/usepagov":1},"maxLength":75,"minLength":19,"unique":7},{"count":7,"frequencies":{"http://epa.gov/ace/americas-children-and-environment-third-edition":1,"http://epa.gov/compliance":1,"http://epa.gov/newsroom":1,"http://epa.gov/pesticides":1,"http://instagram.com/epagov":1,"http://regulations.gov":1,"http://twitter.com/epa":1},"maxLength":66,"minLength":22,"unique":7},{"count":6,"frequencies":{"http://epa.gov/ace/download-graphs-and-data":1,"http://epa.gov/enforcement":1,"http://epa.gov/newsroom/email-subscriptions":1,"http://epa.gov/open":1,"http://epa.gov/radon":1,"http://youtube.com/user/USEPAgov":1},"maxLength":43,"minLength":19,"unique":6},{"count":6,"frequencies":{"http://epa.gov/ace/americas-children-and-environment-third-edition-appendices":1,"http://epa.gov/environmental-topics/science-topics":1,"http://epa.gov/laws-regulations/laws-and-executive-orders":1,"http://flickr.com/photos/usepagov":1,"http://regulations.gov":1,"http://usa.gov":1},"maxLength":77,"minLength":14,"unique":6},{"count":6,"frequencies":{"http://epa.gov/ace/americas-children-and-environment-third-edition-references":1,"http://epa.gov/environmental-topics/water-topics":1,"http://epa.gov/laws-regulations/policy-guidance":1,"http://epa.gov/newsroom/email-subscriptions":1,"http://instagram.com/epagov":1,"http://whitehouse.gov":1},"maxLength":77,"minLength":21,"unique":6},{"count":4,"frequencies":{"http://epa.gov/home/forms/contact-epa":1,"http://epa.gov/laws-regulations/regulations":1,"http://instagram.com/epagov":1,"http://usa.gov":1},"maxLength":43,"minLength":14,"unique":4},{"count":3,"frequencies":{"http://epa.gov/aboutepa":1,"http://epa.gov/home/epa-hotlines":1,"http://whitehouse.gov":1},"maxLength":32,"minLength":21,"unique":3},{"count":3,"frequencies":{"http://epa.gov/aboutepa/epas-administrator":1,"http://epa.gov/foia":1,"http://epa.gov/home/forms/contact-epa":1},"maxLength":42,"minLength":19,"unique":3},{"count":3,"frequencies":{"http://epa.gov/aboutepa/current-epa-leadership":1,"http://epa.gov/home/epa-hotlines":1,"http://epa.gov/home/frequent-questions-specific-epa-programstopics":1},"maxLength":66,"minLength":32,"unique":3},{"count":3,"frequencies":{"http://epa.gov/aboutepa/epa-organization-chart":1,"http://epa.gov/foia":1,"http://facebook.com/EPA":1},"maxLength":46,"minLength":19,"unique":3},{"count":2,"frequencies":{"http://epa.gov/home/frequent-questions-specific-epa-programstopics":1,"http://twitter.com/epa":1},"maxLength":66,"minLength":22,"unique":2},{"count":2,"frequencies":{"http://facebook.com/EPA":1,"http://youtube.com/user/USEPAgov":1},"maxLength":32,"minLength":23,"unique":2},{"count":2,"frequencies":{"http://flickr.com/photos/usepagov":1,"http://twitter.com/epa":1},"maxLength":33,"minLength":22,"unique":2},{"count":2,"frequencies":{"http://instagram.com/epagov":1,"http://youtube.com/user/USEPAgov":1},"maxLength":32,"minLength":27,"unique":2},{"count":1,"frequencies":{"http://flickr.com/photos/usepagov":1},"maxLength":33,"minLength":33,"unique":1},{"count":1,"frequencies":{"http://instagram.com/epagov":1},"maxLength":27,"minLength":27,"unique":1}]},{"count":1,"frequencies":{"http://epa.gov/ace":1},"key":"redirectTo","maxLength":18,"minLength":18,"type":"string","unique":1},{"count":11,"histogram":{"bins":[200,301,302],"frequencies":[10,1]},"key":"status","max":301,"mean":209.1818181818182,"median":301,"min":200,"type":"numeric"},{"count":11,"frequencies":{"2018-03-28T09:18:45.235554272-04:00":1,"2018-03-28T09:25:13.540790419-04:00":1,"2018-03-28T09:25:14.862101674-04:00":1,"2018-03-28T09:25:14.945580151-04:00":1,"2018-03-28T09:25:16.428352736-04:00":1,"2018-03-28T09:25:17.625882413-04:00":1,"2018-03-28T09:25:18.940721061-04:00":1,"2018-03-28T09:25:19.026926128-04:00":1,"2018-03-28T09:25:23.023501668-04:00":1,"2018-03-28T10:16:52.269215284-04:00":1,"2018-03-28T13:48:21.498962156-04:00":1},"key":"timestamp","maxLength":35,"minLength":35,"type":"string","unique":11},{"count":10,"frequencies":{"ACE Biomonitoring | America's Children and the Environment (ACE) | US EPA":1,"America's Children and the Environment (ACE) | US EPA":1,"Contact Us about Section 508 Accessibility | Section 508: Accessibility | US EPA":1,"Frequent Questions about Section 508 | Section 508: Accessibility | US EPA":1,"Learn About Section 508 | Section 508: Accessibility | US EPA":1,"Section 508 Resources | Section 508: Accessibility | US EPA":1,"Section 508 Standards Resources | Section 508: Accessibility | US EPA":1,"Section 508 Standards | Section 508: Accessibility | US EPA":1,"Think 508 First! Section 508 Quick Reference Guide | Section 508: Accessibility | US EPA":1,"What is Section 508? | Section 508: Accessibility | US EPA":1},"key":"title","maxLength":88,"minLength":53,"type":"string","unique":10},{"count":11,"frequencies":{"http://epa.gov/accessibility/forms/contact-us-about-section-508-accessibility":1,"http://epa.gov/accessibility/frequent-questions-about-section-508":1,"http://epa.gov/accessibility/learn-about-section-508":1,"http://epa.gov/accessibility/section-508-resources":1,"http://epa.gov/accessibility/section-508-standards":1,"http://epa.gov/accessibility/section-508-standards-resources":1,"http://epa.gov/accessibility/think-508-first-section-508-quick-reference-guide":1,"http://epa.gov/accessibility/what-section-508":1,"http://epa.gov/ace":1,"http://epa.gov/ace%20":1,"http://epa.gov/ace/ace-biomonitoring":1},"key":"url","maxLength":78,"minLength":18,"type":"string","unique":11}]`)},
	}
	for i, c := range goodCases {
		res := &dataset.Stats{}
		err := m.Stats(&StatsParams{Ref: c.ref}, res)
		if err != nil {
			t.Errorf("%d. case %s: unexpected error: '%s'", i, c.description, err.Error())
			continue
		}
		expect := []interface{}{}
		if err = json.Unmarshal(c.expected, &expect); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expect, res.Stats); diff != "" {
			t.Errorf("%d. '%s' result mismatch (-want +got):%s\n", i, c.description, diff)
		}
	}
}

// Convert the interface value into an array, or panic if not possible
func mustBeArray(i interface{}, err error) []interface{} {
	if err != nil {
		panic(err)
	}
	return i.([]interface{})
}

func TestListRawRefs(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	// TODO(dlong): Put a TestRunner instance here

	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	minute := 0
	dsfs.Timestamp = func() time.Time {
		minute++
		return time.Date(2001, 01, 01, 01, minute, 01, 01, time.UTC)
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)

	var text string
	if err := m.ListRawRefs(&ListParams{}, &text); err != nil {
		t.Fatal(err)
	}

	expect := dstest.Template(t, `0 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      cities
  Path:      {{ .citiesPath }}
  FSIPath:   
  Published: false
1 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      counter
  Path:      {{ .counterPath }}
  FSIPath:   
  Published: false
2 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      craigslist
  Path:      {{ .craigslistPath }}
  FSIPath:   
  Published: false
3 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      movies
  Path:      {{ .moviesPath }}
  FSIPath:   
  Published: false
4 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      sitemap
  Path:      {{ .sitemapPath }}
  FSIPath:   
  Published: false
`, map[string]string{
		"ProfileID":      "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		"citiesPath":     "/mem/QmPWCzaxFoxAu5wS8qXkL6tSA7aR2Lpcwykfz1TbhhpuDp",
		"counterPath":    "/mem/QmVN68yJdLCstVj7YiDjoDvbuxnWKL57D5EAszM7SxtXi3",
		"craigslistPath": "/mem/Qmcph3Wc9LHBGxzt4JVXR4T5ZGD85FQKdMvHWg6aNzqFCD",
		"moviesPath":     "/mem/QmQPS7Nf6dG8zosyAA8zYd64gaLBTAzYsVhMkaMCgCXJST",
		"sitemapPath":    "/mem/QmPk94KBWhGpfSMrEk85fwuFhqfAU84uwrdnwqQf5EV2B5",
	})

	if diff := cmp.Diff(expect, text); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
