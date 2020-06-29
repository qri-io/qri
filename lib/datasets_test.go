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
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
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
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
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
	if err := m.Save(&SaveParams{Private: true}, nil); err == nil {
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
		got := &reporef.DatasetRef{}
		err := m.Save(&c.params, got)
		if err != nil {
			t.Errorf("case %d: '%s' unexpected error: %s", i, c.description, err.Error())
			continue
		}

		if got != nil && c.res != nil {
			expect := c.res.Dataset
			gotDs := got.Dataset
			if err := dataset.CompareDatasets(expect, gotDs); err != nil {
				t.Errorf("case %d ds mistmatch: %s", i, err.Error())
				continue
			}
		}
	}

	bad := []struct {
		description string
		params      SaveParams
		err         string
	}{

		{"empty params", SaveParams{}, "name or bodypath is required"},
		// {&dataset.Dataset{Peername: "foo", Name: "bar"}, nil, "error with previous reference: error fetching peer from store: profile: not found"},
		// {&dataset.Dataset{Peername: "bad", Name: "path", Commit: &dataset.Commit{Qri: "qri:st"}}, nil, "decoding dataset: invalid commit 'qri' value: qri:st"},
		// {&dataset.Dataset{Peername: "bad", Name: "path", BodyPath: "/bad/path"}, nil, "error with previous reference: error fetching peer from store: profile: not found"},
		// {&dataset.Dataset{BodyPath: "testdata/q_bang.svg"}, nil, "invalid data format: unsupported file type: '.svg'"},
		// {&dataset.Dataset{Peername: "me", Name: "cities", BodyPath: "http://localhost:999999/bad/url"}, nil, "fetching body url: Get http://localhost:999999/bad/url: dial tcp: address 999999: invalid port"},
		// {&dataset.Dataset{Name: "bad name", BodyPath: jobsBodyPath}, nil, "invalid name: error: illegal name 'bad name', names must start with a letter and consist of only a-z,0-9, and _. max length 144 characters"},
		// {&dataset.Dataset{BodyPath: jobsBodyPath, Commit: &dataset.Commit{Qri: "qri:st"}}, nil, "decoding dataset: invalid commit 'qri' value: qri:st"},
		{"", SaveParams{Ref: "me/bad", BodyPath: badDataS.URL + "/data.json"}, "determining dataset structure: invalid json data"},
	}

	for i, c := range bad {
		got := &reporef.DatasetRef{}
		err := m.Save(&c.params, got)
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

	res := &reporef.DatasetRef{}
	if err := m.Save(&SaveParams{Ref: ref.Alias()}, res); err == nil {
		t.Error("expected empty save without force flag to error")
	}

	if err := m.Save(&SaveParams{
		Ref:   ref.Alias(),
		Force: true,
	}, res); err != nil {
		t.Errorf("expected empty save with flag to not error. got: %s", err.Error())
	}
}

func TestDatasetRequestsSaveRecallDrop(t *testing.T) {
	t.Skip("TODO(dustmop): Recall will be going away soon, apply will take its place")
	ctx, done := context.WithCancel(context.Background())
	defer done()

	node := newTestQriNode(t)
	ref := addNowTransformDataset(t, node)
	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)

	metaOnePath := tempDatasetFile(t, "*-meta.json", &dataset.Dataset{Meta: &dataset.Meta{Title: "an updated title"}})
	metaTwoPath := tempDatasetFile(t, "*-meta-2.json", &dataset.Dataset{Meta: &dataset.Meta{Title: "new title!"}})
	defer func() {
		os.RemoveAll(metaOnePath)
		os.RemoveAll(metaTwoPath)
	}()

	res := &reporef.DatasetRef{}
	err := m.Save(&SaveParams{
		Ref:        ref.Alias(),
		FilePaths:  []string{metaOnePath},
		ReturnBody: true}, res)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = m.Save(&SaveParams{
		Ref:       ref.Alias(),
		FilePaths: []string{metaOnePath},
		Recall:    "wut"}, res)
	if err == nil {
		t.Fatal("expected bad recall to error")
	}

	err = m.Save(&SaveParams{
		Ref:       ref.Alias(),
		FilePaths: []string{metaTwoPath},
		Recall:    "tf"}, res)
	if err != nil {
		t.Fatal(err)
	}
	if res.Dataset.Transform == nil {
		t.Error("expected transform to exist on recalled save")
	}

	err = m.Save(&SaveParams{
		Ref:  ref.Alias(),
		Drop: "wut",
	}, res)
	if err == nil {
		t.Fatal("expected bad recall to error")
	}

	err = m.Save(&SaveParams{
		Ref:  ref.Alias(),
		Drop: "tf",
	}, res)
	if err != nil {
		t.Fatal("expected bad recall to error")
	}
	if res.Dataset.Transform != nil {
		t.Error("expected transform be nil")
	}
}

func TestDatasetRequestsSaveZip(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)

	res := reporef.DatasetRef{}
	// TODO (b5): import.zip has a ref.txt file that specifies test_user/test_repo as the dataset name,
	// save now requires a string reference. we need to pick a behaviour here & write a test that enforces it
	err = m.Save(&SaveParams{Ref: "me/huh", FilePaths: []string{"testdata/import.zip"}}, &res)
	if err != nil {
		t.Fatal(err.Error())
	}

	if res.Dataset.Commit.Title != "Test Title" {
		t.Fatalf("Expected 'Test Title', got '%s'", res.Dataset.Commit.Title)
	}
	if res.Dataset.Meta.Title != "Test Repo" {
		t.Fatalf("Expected 'Test Repo', got '%s'", res.Dataset.Meta.Title)
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

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
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
		got := []dsref.VersionInfo{}
		err := m.List(c.p, &got)

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

	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
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
			var res []dsref.VersionInfo
			err := m.List(p, &res)
			if err != nil {
				t.Errorf("error listing dataset: %s", err.Error())
			}
			// Get number from end of peername, use that to find dataset name.
			profile, _ := node.Repo.Profile()
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
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)

	ref, err := mr.GetRef(reporef.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Fatalf("error getting path: %s", err.Error())
	}

	moviesDs, err := dsfs.LoadDataset(ctx, mr.Store(), ref.Path)
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
			got := &GetResult{}
			err := dsm.Get(c.params, got)
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
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
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
	got := &GetResult{}
	getParams := &GetParams{
		Refstr: "peer/movies",
	}
	if err := dsm.Get(getParams, got); err != nil {
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

	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
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
			profile, _ := node.Repo.Profile()
			num := profile.Peername[len(profile.Peername)-1:]
			index, _ := strconv.ParseInt(num, 10, 32)
			name := datasets[index]
			ref := reporef.DatasetRef{Peername: profile.Peername, Name: name}

			inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
			m := NewDatasetMethods(inst)
			got := &GetResult{}
			err = m.Get(&GetParams{Refstr: ref.String()}, got)
			if err != nil {
				t.Errorf("error listing dataset for %s: %s", ref.Name, err.Error())
			}

			if got.Bytes == nil {
				t.Errorf("failed to get dataset for %s", ref.Name)
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
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
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
	tr := newTestRunner(t)
	defer tr.Delete()

	workDir := tr.CreateAndChdirToWorkDir("remove_no_history")
	initP := &InitFSIDatasetParams{
		Dir:    workDir,
		Name:   "remove_no_history",
		Format: "csv",
	}
	var refstr string
	if err := NewFSIMethods(tr.Instance).InitDataset(initP, &refstr); err != nil {
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
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
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
	initp := &InitFSIDatasetParams{
		Name:   "no_history",
		Dir:    datasetsDir,
		Format: "csv",
		Mkdir:  "no_history",
	}
	var noHistoryName string
	if err := fsim.InitDataset(initp, &noHistoryName); err != nil {
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
	saveRes := &reporef.DatasetRef{}
	if err := dsm.Save(&SaveParams{Ref: "peer/craigslist", Dataset: &dataset.Dataset{Meta: &dataset.Meta{Title: "oh word"}}}, saveRes); err != nil {
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

func TestDatasetRequestsAdd(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	bad := []struct {
		p   AddParams
		err string
	}{
		{AddParams{Ref: "abc/hash###"}, "node is not online and no registry is configured"},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)
	for i, c := range bad {
		t.Run(fmt.Sprintf("bad_case_%d", i), func(t *testing.T) {
			got := &reporef.DatasetRef{}
			err := m.Add(&c.p, got)
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
	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
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
			m0 := (s0.Repo.Store()).(*cafs.MapStore)
			m1 := (s1.Repo.Store()).(*cafs.MapStore)
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
				profile, _ := p1.Repo.Profile()
				num := profile.Peername[len(profile.Peername)-1:]
				index, _ := strconv.ParseInt(num, 10, 32)
				name := datasets[index]
				ref := reporef.DatasetRef{Peername: profile.Peername, Name: name}
				p := &AddParams{
					Ref: ref.AliasString(),
				}

				// Build requests for peer1 to peer2.
				inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), p0)
				dsm := NewDatasetMethods(inst)
				got := &reporef.DatasetRef{}

				err := dsm.Add(p, got)
				if err != nil {
					pro1, _ := p0.Repo.Profile()
					pro2, _ := p1.Repo.Profile()
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
		p         ValidateDatasetParams
		numErrors int
		err       string
	}{
		{ValidateDatasetParams{Ref: ""}, 0, "bad arguments provided"},
		{ValidateDatasetParams{Ref: "me"}, 0, "cannot find dataset: peer"},
		{ValidateDatasetParams{Ref: "me/movies"}, 4, ""},
		{ValidateDatasetParams{Ref: "me/movies", BodyFilename: bodyFilename}, 1, ""},
		{ValidateDatasetParams{Ref: "me/movies", SchemaFilename: schemaFilename}, 5, ""},
		{ValidateDatasetParams{SchemaFilename: schemaFilename, BodyFilename: bodyFilename}, 1, ""},
	}

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)
	for i, c := range cases {
		got := []jsonschema.KeyError{}
		err := m.Validate(&c.p, &got)
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

func TestDatasetRequestsStats(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
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
		{"empty reference", "", `"" is not a valid dataset reference: empty reference`},
		{"dataset does not exist", "me/dataset_does_not_exist", "reference not found"},
	}
	for i, c := range badCases {
		res := &StatsResponse{}
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
		{"csv: me/cities", "me/cities", []byte(`[{"count":5,"maxLength":8,"minLength":7,"type":"string","unique":5},{"count":5,"histogram":{"bins":[35000,4031500.1,8028000.2,12024500.3,16021000.4,20017500.5,24014000.6,28010500.7,32007000.8,36003500.9,40000001],"frequencies":[3,0,1,0,0,0,0,0,0,1]},"max":40000000,"mean":9817000,"median":300000,"min":35000,"type":"numeric"},{"count":5,"histogram":{"bins":[44.4,46.585,48.769999999999996,50.955,53.14,55.325,57.51,59.695,61.879999999999995,64.065,66.25],"frequencies":[2,0,1,0,0,1,0,0,0,1]},"max":65.25,"mean":52.04,"median":50.65,"min":44.4,"type":"numeric"},{"count":5,"falseCount":1,"trueCount":4,"type":"boolean"}]`)},
		{"json: me/sitemap", "me/sitemap", []byte(`[{"count":10,"histogram":{"bins":[24515,26071.5,27628,29184.5,30741,32297.5,33854,35410.5,36967,38523.5,40080],"frequencies":[4,0,3,1,0,0,1,0,0,1]},"key":"contentLength","max":40079,"mean":28825.8,"median":28059,"min":24515,"type":"numeric"},{"count":10,"frequencies":{"text/html; charset=utf-8":10},"key":"contentSniff","maxLength":24,"minLength":24,"type":"string"},{"count":10,"frequencies":{"text/html; charset=utf-8":10},"key":"contentType","maxLength":24,"minLength":24,"type":"string"},{"count":10,"histogram":{"bins":[74291866,475020463.6,875749061.2,1276477658.8000002,1677206256.4,2077934854,2478663451.6000004,2879392049.2000003,3280120646.8,3680849244.4,4081577842],"frequencies":[2,0,0,0,0,0,0,0,0,8]},"key":"duration","max":4081577841,"mean":3276899953.4,"median":4077230086,"min":74291866,"type":"numeric"},{"count":10,"key":"hash","maxLength":68,"minLength":68,"type":"string","unique":10},{"key":"links","type":"array","values":[{"count":10,"maxLength":58,"minLength":14,"unique":10},{"count":10,"maxLength":115,"minLength":19,"unique":10},{"count":10,"maxLength":68,"minLength":22,"unique":10},{"count":10,"maxLength":115,"minLength":14,"unique":10},{"count":9,"maxLength":70,"minLength":15,"unique":9},{"count":9,"maxLength":115,"minLength":37,"unique":9},{"count":9,"maxLength":52,"minLength":15,"unique":9},{"count":9,"maxLength":75,"minLength":19,"unique":9},{"count":9,"maxLength":66,"minLength":15,"unique":9},{"count":7,"maxLength":75,"minLength":19,"unique":7},{"count":7,"maxLength":66,"minLength":22,"unique":7},{"count":6,"maxLength":43,"minLength":19,"unique":6},{"count":6,"maxLength":77,"minLength":14,"unique":6},{"count":6,"maxLength":77,"minLength":21,"unique":6},{"count":4,"maxLength":43,"minLength":14,"unique":4},{"count":3,"maxLength":32,"minLength":21,"unique":3},{"count":3,"maxLength":42,"minLength":19,"unique":3},{"count":3,"maxLength":66,"minLength":32,"unique":3},{"count":3,"maxLength":46,"minLength":19,"unique":3},{"count":2,"maxLength":66,"minLength":22,"unique":2},{"count":2,"maxLength":32,"minLength":23,"unique":2},{"count":2,"maxLength":33,"minLength":22,"unique":2},{"count":2,"maxLength":32,"minLength":27,"unique":2},{"count":1,"maxLength":33,"minLength":33,"unique":1},{"count":1,"maxLength":27,"minLength":27,"unique":1}]},{"count":1,"key":"redirectTo","maxLength":18,"minLength":18,"type":"string","unique":1},{"count":11,"histogram":{"bins":[200,210.2,220.4,230.6,240.8,251,261.2,271.4,281.6,291.8,302],"frequencies":[10,0,0,0,0,0,0,0,0,1]},"key":"status","max":301,"mean":209.1818181818182,"median":200,"min":200,"type":"numeric"},{"count":11,"key":"timestamp","maxLength":35,"minLength":35,"type":"string","unique":11},{"count":10,"key":"title","maxLength":88,"minLength":53,"type":"string","unique":10},{"count":11,"key":"url","maxLength":78,"minLength":18,"type":"string","unique":11}]`)},
	}
	for i, c := range goodCases {
		res := &StatsResponse{}
		err := m.Stats(&StatsParams{Ref: c.ref}, res)
		if err != nil {
			t.Errorf("%d. case %s: unexpected error: '%s'", i, c.description, err.Error())
			continue
		}
		if diff := cmp.Diff(c.expected, res.StatsBytes); diff != "" {
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
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, config.DefaultConfigForTesting(), node)
	m := NewDatasetMethods(inst)

	var text string
	if err := m.ListRawRefs(&ListParams{}, &text); err != nil {
		t.Fatal(err)
	}
	expect := `0 Peername:  peer
  ProfileID: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
  Name:      cities
  Path:      /map/QmcvCpP2qBYV4szUV4MkJopaZM6u8kbsYfhZxbk91ZY5s5
  FSIPath:   
  Published: false
1 Peername:  peer
  ProfileID: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
  Name:      counter
  Path:      /map/QmTve65WAZJqg6gGPU4o4YQSVdQ11bTA5YaFjyK9mnEQ48
  FSIPath:   
  Published: false
2 Peername:  peer
  ProfileID: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
  Name:      craigslist
  Path:      /map/QmX1sNkK7PYfx9344vcd6vK8fYRBV5NyH7C4Wqz6Y2x4zX
  FSIPath:   
  Published: false
3 Peername:  peer
  ProfileID: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
  Name:      movies
  Path:      /map/QmVnSLjFfZ8QRyTvAMfqjWoTZS1zo4JthKjBaFjnneirAc
  FSIPath:   
  Published: false
4 Peername:  peer
  ProfileID: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
  Name:      sitemap
  Path:      /map/QmdDVuAMJLCSQ4rCemBVh1KUk3cb7jSaCvXt9n2X75gnGj
  FSIPath:   
  Published: false
`
	if diff := cmp.Diff(expect, text); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
