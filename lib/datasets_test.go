package lib

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/dsdiff"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
	"github.com/qri-io/qri/rev"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func init() {
	dsfs.Timestamp = func() time.Time {
		return time.Time{}
	}
}

func TestDatasetRequestsSave(t *testing.T) {
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	citiesBodyPath, err := dstest.BodyFilepath("testdata/cities_2")
	if err != nil {
		t.Fatal(err.Error())
	}
	// TODO: Needed for TestCases for `new`, see below.
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

	req := NewDatasetRequests(node, nil)

	privateErrMsg := "option to make dataset private not yet implimented, refer to https://github.com/qri-io/qri/issues/291 for updates"
	if err := req.Save(&SaveParams{Private: true}, nil); err == nil {
		t.Errorf("expected datset to error")
	} else if err.Error() != privateErrMsg {
		t.Errorf("private flag error mismatch: expected: '%s', got: '%s'", privateErrMsg, err.Error())
	}

	cases := []struct {
		dataset *dataset.Dataset
		res     *dataset.Dataset
		err     string
	}{

		// {&dataset.Dataset{
		// 	Structure: &dataset.StructurePod{Schema: map[string]interface{}{"type": "string"}},
		// 	BodyPath:  jobsBodyPath,
		// }, nil, "invalid dataset: structure: format is required"},
		// {&dataset.Dataset{BodyPath: jobsBodyPath, Commit: &dataset.Commit{}}, nil, ""},

		// {nil, nil, "at least one of Dataset, DatasetPath is required"},
		// TODO - restore
		{&dataset.Dataset{}, nil, "name is required"},
		// {&dataset.Dataset{Peername: "foo", Name: "bar"}, nil, "error with previous reference: error fetching peer from store: profile: not found"},
		// {&dataset.Dataset{Peername: "bad", Name: "path", Commit: &dataset.Commit{Qri: "qri:st"}}, nil, "decoding dataset: invalid commit 'qri' value: qri:st"},
		// {&dataset.Dataset{Peername: "bad", Name: "path", BodyPath: "/bad/path"}, nil, "error with previous reference: error fetching peer from store: profile: not found"},
		// {&dataset.Dataset{BodyPath: "testdata/q_bang.svg"}, nil, "invalid data format: unsupported file type: '.svg'"},
		// {&dataset.Dataset{Peername: "me", Name: "cities", BodyPath: "http://localhost:999999/bad/url"}, nil, "fetching body url: Get http://localhost:999999/bad/url: dial tcp: address 999999: invalid port"},
		// {&dataset.Dataset{Name: "bad name", BodyPath: jobsBodyPath}, nil, "invalid name: error: illegal name 'bad name', names must start with a letter and consist of only a-z,0-9, and _. max length 144 characters"},
		// {&dataset.Dataset{BodyPath: jobsBodyPath, Commit: &dataset.Commit{Qri: "qri:st"}}, nil, "decoding dataset: invalid commit 'qri' value: qri:st"},
		{&dataset.Dataset{Peername: "me", Name: "bad", BodyPath: badDataS.URL + "/data.json"}, nil, "determining dataset structure: invalid json data"},
		{&dataset.Dataset{Name: "jobs_ranked_by_automation_prob", BodyPath: jobsBodyPath}, nil, ""},
		{&dataset.Dataset{Peername: "me", Name: "cities", Meta: &dataset.Meta{Title: "updated name of movies dataset"}}, nil, ""},
		{&dataset.Dataset{Peername: "me", Name: "cities", Commit: &dataset.Commit{}, BodyPath: citiesBodyPath}, nil, ""},
		{&dataset.Dataset{Peername: "me", Name: "cities", BodyPath: s.URL + "/body.csv"}, nil, ""},
	}

	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Save(&SaveParams{Dataset: c.dataset}, got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Fatalf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			// t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if got != nil && c.res != nil {
			expect := c.res
			// expect := &dataset.Dataset{}
			// if err := expect.Decode(c.res); err != nil {
			// 	t.Errorf("case %d error decoding expect dataset: %s", i, err.Error())
			// 	continue
			// }
			gotDs := got.Dataset
			// if err := gotDs.Decode(got.Dataset); err != nil {
			// 	t.Errorf("case %d error decoding got dataset: %s", i, err.Error())
			// 	continue
			// }
			if err := dataset.CompareDatasets(expect, gotDs); err != nil {
				t.Errorf("case %d ds mistmatch: %s", i, err.Error())
				continue
			}
		}
	}
}

func TestDatasetRequestsSaveRecall(t *testing.T) {
	node := newTestQriNode(t)
	ref := addNowTransformDataset(t, node)
	r := NewDatasetRequests(node, nil)

	res := &repo.DatasetRef{}
	err := r.Save(&SaveParams{Dataset: &dataset.Dataset{
		Peername: ref.Peername,
		Name:     ref.Name,
		Meta:     &dataset.Meta{Title: "an updated title"},
	}, ReturnBody: true}, res)
	if err != nil {
		t.Error("save failed")
	}

	err = r.Save(&SaveParams{
		Dataset: &dataset.Dataset{
			Peername: ref.Peername,
			Name:     ref.Name,
			Meta:     &dataset.Meta{Title: "an updated title"},
		},
		Recall: "wut"}, res)
	if err == nil {
		t.Error("expected bad recall to error")
	}

	err = r.Save(&SaveParams{
		Dataset: &dataset.Dataset{
			Peername: ref.Peername,
			Name:     ref.Name,
			Meta:     &dataset.Meta{Title: "new title!"},
		},
		Recall: "tf"}, res)
	if err != nil {
		t.Error(err)
	}
	if res.Dataset.Transform == nil {
		t.Error("expected transform to exist on recalled save")
	}
}

func TestDatasetRequestsSaveZip(t *testing.T) {
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	req := NewDatasetRequests(node, nil)

	dsp := &dataset.Dataset{Peername: "me"}
	res := repo.DatasetRef{}
	err = req.Save(&SaveParams{Dataset: dsp, DatasetPath: "testdata/import.zip"}, &res)
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

func TestDatasetRequestsUpdate(t *testing.T) {
	node := newTestQriNode(t)

	r := NewDatasetRequests(node, nil)
	res := &repo.DatasetRef{}
	if err := r.Update(&UpdateParams{Ref: "me/bad_dataset"}, res); err == nil {
		t.Error("expected update to nonexistent dataset to error")
	}

	ref := addNowTransformDataset(t, node)
	res = &repo.DatasetRef{}
	if err := r.Update(&UpdateParams{Ref: ref.AliasString(), Recall: "tf", ReturnBody: true}, res); err != nil {
		t.Errorf("update error: %s", err)
	}

	// run a manual save to lose the transform
	err := r.Save(&SaveParams{Dataset: &dataset.Dataset{
		Peername: res.Peername,
		Name:     res.Name,
		Meta:     &dataset.Meta{Title: "an updated title"},
	}}, res)
	if err != nil {
		t.Error("save failed")
	}

	if err := r.Update(&UpdateParams{Ref: res.AliasString(), ReturnBody: true}, res); err == nil {
		t.Error("expected updated without transform to error")
	}
}

func TestDatasetRequestsList(t *testing.T) {
	var (
		movies, counter, cities, craigslist, sitemap repo.DatasetRef
	)

	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
		return
	}

	refs, err := mr.References(30, 0)
	if err != nil {
		t.Fatalf("error getting namespace: %s", err.Error())
	}

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
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
		case "sitemap":
			sitemap = ref
		}
	}

	cases := []struct {
		p   *ListParams
		res []repo.DatasetRef
		err string
	}{
		{&ListParams{OrderBy: "", Limit: 1, Offset: 0}, nil, ""},
		{&ListParams{OrderBy: "chaos", Limit: 1, Offset: -50}, nil, ""},
		{&ListParams{OrderBy: "", Limit: 30, Offset: 0}, []repo.DatasetRef{cities, counter, craigslist, movies, sitemap}, ""},
		{&ListParams{OrderBy: "timestamp", Limit: 30, Offset: 0}, []repo.DatasetRef{cities, counter, craigslist, movies, sitemap}, ""},
		{&ListParams{Peername: "me", OrderBy: "timestamp", Limit: 30, Offset: 0}, []repo.DatasetRef{cities, counter, craigslist, movies, sitemap}, ""},
		// TODO: re-enable {&ListParams{OrderBy: "name", Limit: 30, Offset: 0}, []*repo.DatasetRef{cities, counter, movies}, ""},
	}

	req := NewDatasetRequests(node, nil)
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

func TestDatasetRequestsListP2p(t *testing.T) {
	// Matches what is used to generated test peers.
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	ctx := context.Background()
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

			dsr := NewDatasetRequests(node, nil)
			p := &ListParams{OrderBy: "", Limit: 30, Offset: 0}
			var res []repo.DatasetRef
			err := dsr.List(p, &res)
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
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	ref, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Fatalf("error getting path: %s", err.Error())
	}

	moviesDs, err := dsfs.LoadDataset(mr.Store(), ref.Path)
	if err != nil {
		t.Fatalf("error loading dataset: %s", err.Error())
	}

	cases := []struct {
		p   repo.DatasetRef
		res *dataset.Dataset
		err string
	}{
		// TODO: probably delete some of these
		{repo.DatasetRef{Peername: "peer", Path: "abc", Name: "ABC"}, nil,
			"error loading dataset: error getting file bytes: cafs: path not found"},
		{repo.DatasetRef{Peername: "peer", Path: ref.Path, Name: "ABC"}, nil, ""},
		{repo.DatasetRef{Peername: "peer", Path: ref.Path, Name: "movies"}, moviesDs, ""},
		{repo.DatasetRef{Peername: "peer", Path: ref.Path, Name: "cats"}, moviesDs, ""},
	}

	req := NewDatasetRequests(node, nil)
	for i, c := range cases {
		got := &LookupResult{}
		err := req.Get(&LookupParams{Ref: &c.p}, got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
		// TODO (dlong): Inspect the contents of `got`
	}
}

func TestDatasetRequestsGetP2p(t *testing.T) {
	// Matches what is used to generated test peers.
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	ctx := context.Background()
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
			ref := repo.DatasetRef{Peername: profile.Peername, Name: name}

			dsr := NewDatasetRequests(node, nil)
			got := &LookupResult{}
			err = dsr.Get(&LookupParams{Ref: &ref}, got)
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
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	cases := []struct {
		p   *RenameParams
		res string
		err string
	}{
		{&RenameParams{}, "", "current name is required to rename a dataset"},
		{&RenameParams{Current: repo.DatasetRef{Peername: "peer", Name: "movies"}, New: repo.DatasetRef{Peername: "peer", Name: "new movies"}}, "", "error: illegal name 'new movies', names must start with a letter and consist of only a-z,0-9, and _. max length 144 characters"},
		{&RenameParams{Current: repo.DatasetRef{Peername: "peer", Name: "movies"}, New: repo.DatasetRef{Peername: "peer", Name: "new_movies"}}, "new_movies", ""},
		{&RenameParams{Current: repo.DatasetRef{Peername: "peer", Name: "cities"}, New: repo.DatasetRef{Peername: "peer", Name: "sitemap"}}, "", "dataset 'peer/sitemap' already exists"},
	}

	req := NewDatasetRequests(node, nil)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Rename(c.p, got)

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
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	ref, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Fatalf("error getting movies ref: %s", err.Error())
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

	req := NewDatasetRequests(node, nil)
	for i, c := range cases {
		numDeleted := 0
		params := RemoveParams{Ref: c.p, Revision: rev.Rev{Field: "ds", Gen: -1}}
		err := req.Remove(&params, &numDeleted)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestDatasetRequestsLookupBody(t *testing.T) {
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
		return
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	moviesRef, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "movies"})
	if err != nil {
		t.Fatalf("error getting movies ref: %s", err.Error())
	}
	clRef, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "craigslist"})
	if err != nil {
		t.Fatalf("error getting craigslist ref: %s", err.Error())
	}
	sitemapRef, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "sitemap"})
	if err != nil {
		t.Fatalf("error getting sitemap ref: %s", err.Error())
	}

	cases := []struct {
		p        *LookupParams
		resCount int
		err      string
	}{
		{&LookupParams{}, 0, "repo: empty dataset reference"},
		{&LookupParams{Format: "json", PathString: moviesRef.Path, Limit: 5, Offset: 0, All: false}, 5, ""},
		{&LookupParams{Format: "json", PathString: moviesRef.Path, Limit: -5, Offset: -100, All: false}, 0, "invalid limit / offset settings"},
		{&LookupParams{Format: "json", PathString: moviesRef.Path, Limit: -5, Offset: -100, All: true}, 0, "invalid limit / offset settings"},
		{&LookupParams{Format: "json", PathString: clRef.Path, Limit: 0, Offset: 0, All: true}, 0, ""},
		{&LookupParams{Format: "json", PathString: clRef.Path, Limit: 2, Offset: 0, All: false}, 2, ""},
		{&LookupParams{Format: "json", PathString: sitemapRef.Path, Limit: 3, Offset: 0, All: false}, 3, ""},
	}

	req := NewDatasetRequests(node, nil)
	for i, c := range cases {
		got := &LookupResult{}
		err := req.LookupBody(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if got.Bytes == nil && c.resCount == 0 {
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
		case "csv":
			r := csv.NewReader(bytes.NewBuffer(got.Bytes))
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
		{&repo.DatasetRef{Name: "abc", Path: "hash###"}, nil, "node is not online and no registry is configured"},
	}

	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	req := NewDatasetRequests(node, nil)
	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Add(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}

func TestDatasetRequestsAddP2P(t *testing.T) {
	// Matches what is used to generate the test peers.
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	// Create test nodes.
	ctx := context.Background()
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
				ref := repo.DatasetRef{Peername: profile.Peername, Name: name}

				// Build requests for peer1 to peer2.
				dsr := NewDatasetRequests(p0, nil)
				got := &repo.DatasetRef{}

				err := dsr.Add(&ref, got)
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
		{ValidateDatasetParams{Ref: repo.DatasetRef{}}, 0, "bad arguments provided"},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me"}}, 0, "cannot find dataset: peer@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me", Name: "movies"}}, 4, ""},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me", Name: "movies"}, Data: dataf, DataFilename: "data.csv"}, 1, ""},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me", Name: "movies"}, Schema: schemaf}, 4, ""},
		{ValidateDatasetParams{Schema: schemaf2, DataFilename: "data.csv", Data: dataf2}, 1, ""},
	}

	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	req := NewDatasetRequests(node, nil)
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

func TestDatasetRequestsDiff(t *testing.T) {
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	req := NewDatasetRequests(node, nil)

	// File 1
	fp1, err := dstest.BodyFilepath("testdata/jobs_by_automation")
	if err != nil {
		t.Errorf("getting data filepath: %s", err.Error())
		return
	}

	dsRef1 := repo.DatasetRef{}
	initParams := &SaveParams{
		Dataset: &dataset.Dataset{
			Name:     "jobs_ranked_by_automation_prob",
			BodyPath: fp1,
		},
	}

	err = req.Save(initParams, &dsRef1)
	if err != nil {
		t.Errorf("couldn't init file 1: %s", err.Error())
		return
	}

	// File 2
	fp2, err := dstest.BodyFilepath("testdata/jobs_by_automation_2")
	if err != nil {
		t.Errorf("getting data filepath: %s", err.Error())
		return
	}
	dsRef2 := repo.DatasetRef{}
	initParams = &SaveParams{
		Dataset: &dataset.Dataset{
			Name:     "jobs_ranked_by_automation_prob",
			BodyPath: fp2,
		},
	}
	err = req.Save(initParams, &dsRef2)
	if err != nil {
		t.Errorf("couldn't load second file: %s", err.Error())
		return
	}

	//test cases
	cases := []struct {
		Left, Right   repo.DatasetRef
		All           bool
		Components    map[string]bool
		displayFormat string
		expected      string
		err           string
	}{
		{dsRef1, dsRef2, false, map[string]bool{"structure": true}, "listKeys", "Structure: 3 changes\n\t- modified checksum\n\t- modified length\n\t- modified schema", ""},
		{dsRef1, dsRef2, true, nil, "listKeys", "Structure: 3 changes\n\t- modified checksum\n\t- modified length\n\t- modified schema", ""},
	}
	// execute
	for i, c := range cases {
		p := &DiffParams{
			Left:           c.Left,
			Right:          c.Right,
			DiffAll:        c.All,
			DiffComponents: c.Components,
		}
		res := map[string]*dsdiff.SubDiff{}
		err := req.Diff(p, &res)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected '%s', got '%s'", i, c.err, err.Error())
		}

		if c.err != "" {
			continue
		}

		stringDiffs, err := dsdiff.MapDiffsToString(res, c.displayFormat)
		if err != nil {
			t.Errorf("case %d error mapping to string: %s", i, err.Error())
		}
		if stringDiffs != c.expected {
			t.Errorf("case %d response mistmatch: expected '%s', got '%s'", i, c.expected, stringDiffs)
		}
	}
}
