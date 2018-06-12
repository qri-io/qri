package core

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

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/dsdiff"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
	regmock "github.com/qri-io/registry/regserver/mock"
	"github.com/qri-io/skytf"
)

func init() {
	dsfs.Timestamp = func() time.Time {
		return time.Time{}
	}
}

func TestDatasetRequestsInit(t *testing.T) {
	jobsDataPath, err := dstest.DataFilepath("testdata/jobs_by_automation")
	if err != nil {
		t.Error(err.Error())
		return
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"json":"data"}`))
	}))
	badDataS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`\\\{"json":"data"}`))
	}))

	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewDatasetRequests(mr, nil)

	privateErrMsg := "option to make dataset private not yet implimented, refer to https://github.com/qri-io/qri/issues/291 for updates"
	if err := req.Init(&SaveParams{Private: true}, nil); err == nil {
		t.Errorf("expected datset to error")
	} else if err.Error() != privateErrMsg {
		t.Errorf("private flag error mismatch: expected: '%s', got: '%s'", privateErrMsg, err.Error())
	}

	cases := []struct {
		dataset *dataset.DatasetPod
		res     *dataset.DatasetPod
		err     string
	}{
		{nil, nil, "dataset is required"},
		{&dataset.DatasetPod{}, nil, "either dataBytes, dataPath, or a transform is required to create a dataset"},
		{&dataset.DatasetPod{DataPath: "/bad/path"}, nil, "opening file: open /bad/path: no such file or directory"},
		{&dataset.DatasetPod{DataPath: jobsDataPath, Commit: &dataset.CommitPod{Qri: "qri:st"}}, nil, "decoding dataset: invalid commit 'qri' value: qri:st"},
		{&dataset.DatasetPod{DataPath: "http://localhost:999999/bad/url"}, nil, "fetching data url: Get http://localhost:999999/bad/url: dial tcp: address 999999: invalid port"},
		{&dataset.DatasetPod{Name: "bad name", DataPath: jobsDataPath}, nil, "invalid name: error: illegal name 'bad name', names must start with a letter and consist of only a-z,0-9, and _. max length 144 characters"},
		{&dataset.DatasetPod{DataPath: badDataS.URL + "/data.json"}, nil, "determining dataset schema: invalid json data"},
		{&dataset.DatasetPod{DataPath: "testdata/q_bang.svg"}, nil, "invalid data format: unsupported file type: '.svg'"},

		{&dataset.DatasetPod{
			Structure: &dataset.StructurePod{Schema: map[string]interface{}{"type": "string"}},
			DataPath:  jobsDataPath,
		}, nil, "invalid dataset: structure: format is required"},
		{&dataset.DatasetPod{DataPath: jobsDataPath, Commit: &dataset.CommitPod{}}, nil, ""},
		{&dataset.DatasetPod{DataPath: s.URL + "/data.json"}, nil, ""},

		// confirm input metadata overwrites transform metadata
		{&dataset.DatasetPod{
			Name: "foo",
			Meta: &dataset.Meta{Title: "foo"},
			Transform: &dataset.TransformPod{
				ScriptPath: "testdata/tf/transform.sky",
			}},
			&dataset.DatasetPod{
				Name:     "foo",
				Qri:      "qri:ds:0",
				DataPath: "/map/QmYMHqqgzR2V1sMD6g68EDPSZrpsvY6zZM22TagzZVxiKQ",
				Commit: &dataset.CommitPod{
					Qri:       "cm:0",
					Title:     "created dataset",
					Signature: "Ak4bJBNUt+XWH2xTiY4Da4I5eZmtsTZMhNS6f4Sb0cgYrrsuOCTQ3NJUlbYR9gyyYiXp3p+pgV8JmzYnUJkllFL6g00Bc0CkNC+N0/pkONYJY180BxPmqzz7oZEBqpLEtqzOP7QsaXFSXBqBkVAJxGBiLj7WunECGVayeYeKoWRcNnZCeW1y8LiDmiP2CahzbievFQs0fyFI3c9hJqxFq0YEjMzOCG9ICejOtitJkAwjLOO46mS4XC0enCRlkqtpPwnTD0dWtfmbx7+laxJJlbxx1wpnyjsDnupua7UB+GS9V7QCZDl915IF/sLCfRJ5j/PNCdmndgtl5/otJHum7Q==",
				},
				Meta: &dataset.Meta{Qri: "md:0", Title: "foo"},
				Transform: &dataset.TransformPod{
					Qri:           "tf:0",
					Syntax:        "skylark",
					SyntaxVersion: skytf.Version,
					ScriptPath:    "/map/QmcjVAiafyztY4rKjmZQZQybMMEo9EPekSErzs6eHtadfg",
				},
				Structure: &dataset.StructurePod{
					Qri:      "st:0",
					Format:   dataset.JSONDataFormat.String(),
					Length:   17,
					Entries:  2,
					Checksum: "QmYMHqqgzR2V1sMD6g68EDPSZrpsvY6zZM22TagzZVxiKQ",
					Schema: map[string]interface{}{
						"type": "array",
					},
				},
			},
			""},
	}

	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Init(&SaveParams{Dataset: c.dataset, Publish: true}, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if got != nil && c.res != nil {
			expect := &dataset.Dataset{}
			if err := expect.Decode(c.res); err != nil {
				t.Errorf("case %d error decoding expect dataset: %s", i, err.Error())
				continue
			}
			gotDs := &dataset.Dataset{}
			if err := gotDs.Decode(got.Dataset); err != nil {
				t.Errorf("case %d error decoding got dataset: %s", i, err.Error())
				continue
			}
			if err := dataset.CompareDatasets(expect, gotDs); err != nil {
				t.Errorf("case %d ds mistmatch: %s", i, err.Error())
				continue
			}
		}
	}
}

func TestDatasetRequestsSave(t *testing.T) {
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	citiesDataPath, err := dstest.DataFilepath("testdata/cities_2")
	if err != nil {
		t.Error(err.Error())
		return
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

	req := NewDatasetRequests(mr, nil)

	privateErrMsg := "option to make dataset private not yet implimented, refer to https://github.com/qri-io/qri/issues/291 for updates"
	if err := req.Save(&SaveParams{Private: true}, nil); err == nil {
		t.Errorf("expected datset to error")
	} else if err.Error() != privateErrMsg {
		t.Errorf("private flag error mismatch: expected: '%s', got: '%s'", privateErrMsg, err.Error())
	}

	cases := []struct {
		dataset *dataset.DatasetPod
		res     *dataset.DatasetPod
		err     string
	}{
		{nil, nil, "dataset is required"},
		{&dataset.DatasetPod{}, nil, "peername & name are required to update dataset"},
		{&dataset.DatasetPod{Peername: "foo", Name: "bar"}, nil, "canonicalizing previous dataset reference: error fetching peer from store: profile: not found"},
		{&dataset.DatasetPod{Peername: "bad", Name: "path", Commit: &dataset.CommitPod{Qri: "qri:st"}}, nil, "decoding dataset: invalid commit 'qri' value: qri:st"},
		{&dataset.DatasetPod{Peername: "bad", Name: "path", DataPath: "/bad/path"}, nil, "canonicalizing previous dataset reference: error fetching peer from store: profile: not found"},
		{&dataset.DatasetPod{Peername: "me", Name: "cities", DataPath: "http://localhost:999999/bad/url"}, nil, "fetching data url: Get http://localhost:999999/bad/url: dial tcp: address 999999: invalid port"},

		{&dataset.DatasetPod{Peername: "me", Name: "cities", Meta: &dataset.Meta{Title: "updated name of movies dataset"}}, nil, ""},
		{&dataset.DatasetPod{Peername: "me", Name: "cities", Commit: &dataset.CommitPod{}, DataPath: citiesDataPath}, nil, ""},
		{&dataset.DatasetPod{Peername: "me", Name: "cities", DataPath: s.URL + "/data.csv"}, nil, ""},
	}

	for i, c := range cases {
		got := &repo.DatasetRef{}
		err := req.Save(&SaveParams{Dataset: c.dataset}, got)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}

		if got != nil && c.res != nil {
			expect := &dataset.Dataset{}
			if err := expect.Decode(c.res); err != nil {
				t.Errorf("case %d error decoding expect dataset: %s", i, err.Error())
				continue
			}
			gotDs := &dataset.Dataset{}
			if err := gotDs.Decode(got.Dataset); err != nil {
				t.Errorf("case %d error decoding got dataset: %s", i, err.Error())
				continue
			}
			if err := dataset.CompareDatasets(expect, gotDs); err != nil {
				t.Errorf("case %d ds mistmatch: %s", i, err.Error())
				continue
			}
		}
	}
}

func TestDatasetRequestsList(t *testing.T) {
	var (
		movies, counter, cities, craigslist, sitemap repo.DatasetRef
	)

	mr, err := testrepo.NewTestRepo(nil)
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

func TestDatasetRequestsListP2p(t *testing.T) {
	// Matches what is used to generated test peers.
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	ctx := context.Background()
	testPeers, err := p2ptest.NewTestNetwork(ctx, t, 5, p2p.NewTestQriNode)
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

			dsr := NewDatasetRequestsWithNode(node.Repo, nil, node)
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
		// TODO: probably delete some of these
		{repo.DatasetRef{Peername: "peer", Path: "abc", Name: "ABC"}, nil, "repo: not found"},
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

func TestDatasetRequestsGetP2p(t *testing.T) {
	// Matches what is used to generated test peers.
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	ctx := context.Background()
	testPeers, err := p2ptest.NewTestNetwork(ctx, t, 5, p2p.NewTestQriNode)
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
			profile, _ := node.Repo.Profile()
			num := profile.Peername[len(profile.Peername)-1:]
			index, _ := strconv.ParseInt(num, 10, 32)
			name := datasets[index]
			ref := repo.DatasetRef{Peername: profile.Peername, Name: name}

			dsr := NewDatasetRequestsWithNode(node.Repo, nil, node)
			got := &repo.DatasetRef{}
			err = dsr.Get(&ref, got)
			if err != nil {
				t.Errorf("error listing dataset for %s: %s", ref.Name, err.Error())
			}

			if got.Dataset == nil {
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

func TestDatasetRequestsLookupBody(t *testing.T) {
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
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
	sitemapRef, err := mr.GetRef(repo.DatasetRef{Peername: "peer", Name: "sitemap"})
	if err != nil {
		t.Errorf("error getting sitemap ref: %s", err.Error())
		return
	}

	var df1 = dataset.JSONDataFormat
	cases := []struct {
		p        *LookupParams
		resCount int
		err      string
	}{
		{&LookupParams{}, 0, "error loading dataset: error getting file bytes: datastore: key not found"},
		{&LookupParams{Format: df1, Path: moviesRef.Path, Limit: 5, Offset: 0, All: false}, 5, ""},
		{&LookupParams{Format: df1, Path: moviesRef.Path, Limit: -5, Offset: -100, All: false}, 0, "invalid limit / offset settings"},
		{&LookupParams{Format: df1, Path: moviesRef.Path, Limit: -5, Offset: -100, All: true}, 0, "invalid limit / offset settings"},
		{&LookupParams{Format: df1, Path: clRef.Path, Limit: 0, Offset: 0, All: true}, 0, ""},
		{&LookupParams{Format: df1, Path: clRef.Path, Limit: 2, Offset: 0, All: false}, 2, ""},
		{&LookupParams{Format: df1, Path: sitemapRef.Path, Limit: 3, Offset: 0, All: false}, 3, ""},
	}

	req := NewDatasetRequests(mr, nil)
	for i, c := range cases {
		got := &LookupResult{}
		err := req.LookupBody(c.p, got)

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
		{&repo.DatasetRef{Name: "abc", Path: "hash###"}, nil, "error fetching file: this store cannot fetch from remote sources"},
	}

	mr, err := testrepo.NewTestRepo(nil)
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

func TestDatasetRequestsAddP2p(t *testing.T) {
	// Matches what is used to generate the test peers.
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	// Create test nodes.
	ctx := context.Background()
	testPeers, err := p2ptest.NewTestNetwork(ctx, t, 2, p2p.NewTestQriNode)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	// Peers exchange Qri profile information.
	if err := p2ptest.ConnectQriPeers(ctx, testPeers); err != nil {
		t.Errorf("error connecting peers: %s", err.Error())
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
				dsr := NewDatasetRequestsWithNode(p0.Repo, nil, p0)
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
		{ValidateDatasetParams{Ref: repo.DatasetRef{}}, 0, "either data or a dataset reference is required"},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me"}}, 0, "cannot find dataset: peer@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me", Name: "movies"}}, 15, ""},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me", Name: "movies"}, Data: dataf, DataFilename: "data.csv"}, 1, ""},
		{ValidateDatasetParams{Ref: repo.DatasetRef{Peername: "me", Name: "movies"}, Schema: schemaf}, 15, ""},
		{ValidateDatasetParams{Schema: schemaf2, DataFilename: "data.csv", Data: dataf2}, 1, ""},
	}

	mr, err := testrepo.NewTestRepo(nil)
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

func TestDatasetRequestsDiff(t *testing.T) {
	rc, _ := regmock.NewMockServer()
	mr, err := testrepo.NewTestRepo(rc)
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}
	req := NewDatasetRequests(mr, nil)

	// File 1
	fp1, err := dstest.DataFilepath("testdata/jobs_by_automation")
	if err != nil {
		t.Errorf("getting data filepath: %s", err.Error())
		return
	}

	dsRef1 := repo.DatasetRef{}
	initParams := &SaveParams{
		Dataset: &dataset.DatasetPod{
			Name:     "jobs_ranked_by_automation_prob",
			DataPath: fp1,
		},
	}
	err = req.Init(initParams, &dsRef1)
	if err != nil {
		t.Errorf("couldn't init file 1: %s", err.Error())
		return
	}

	// File 2
	fp2, err := dstest.DataFilepath("testdata/jobs_by_automation_2")
	if err != nil {
		t.Errorf("getting data filepath: %s", err.Error())
		return
	}
	dsRef2 := repo.DatasetRef{}
	initParams = &SaveParams{
		Dataset: &dataset.DatasetPod{
			Name:     "jobs_ranked_by_automation_prob",
			DataPath: fp2,
		},
	}
	err = req.Init(initParams, &dsRef2)
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
