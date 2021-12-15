package lib

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/base/params"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	reporef "github.com/qri-io/qri/repo/ref"
	testrepo "github.com/qri-io/qri/repo/test"
)

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

	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err)
	}

	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

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
		p           *CollectionListParams
		res         []dsref.VersionInfo
		err         string
	}{
		{"list datasets - empty (default)", &CollectionListParams{}, []dsref.VersionInfo{cities, counter, craigslist, movies, sitemap}, ""},
		{"list datasets - weird", &CollectionListParams{List: params.List{Limit: -33, Offset: -50}}, []dsref.VersionInfo{}, "limit of -33 is out of bounds"},
		{"list datasets - happy path", &CollectionListParams{List: params.List{Limit: 30, Offset: 0}}, []dsref.VersionInfo{cities, counter, craigslist, movies, sitemap}, ""},
		{"list datasets - limit 2 offset 0", &CollectionListParams{List: params.List{Limit: 2, Offset: 0}}, []dsref.VersionInfo{cities, counter}, ""},
		{"list datasets - limit 2 offset 2", &CollectionListParams{List: params.List{Limit: 2, Offset: 2}}, []dsref.VersionInfo{craigslist, movies}, ""},
		{"list datasets - limit 2 offset 4", &CollectionListParams{List: params.List{Limit: 2, Offset: 4}}, []dsref.VersionInfo{sitemap}, ""},
		{"list datasets - limit 2 offset 5", &CollectionListParams{List: params.List{Limit: 2, Offset: 5}}, []dsref.VersionInfo{}, ""},
		{"list datasets - order by timestamp", &CollectionListParams{List: params.List{OrderBy: params.OrderBy{{Key: "timestamp", Direction: params.OrderDESC}}, Limit: 30, Offset: 0}}, []dsref.VersionInfo{cities, counter, craigslist, movies, sitemap}, ""},
		{"list datasets - username 'me'", &CollectionListParams{Username: "me", List: params.List{OrderBy: params.OrderBy{{Key: "timestamp", Direction: params.OrderDESC}}, Limit: 30, Offset: 0}}, []dsref.VersionInfo{cities, counter, craigslist, movies, sitemap}, ""},
		// TODO: re-enable {&CollectionListParams{List: params.List {OrderBy: params.OrderBy{{Key: "name", Direction: OrderASC }}, Limit: 30, Offset: 0}}, []*dsref.VersionInfo{cities, counter, movies}, ""}},
	}

	for _, c := range cases {
		got, _, err := inst.Collection().List(ctx, c.p)

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
		return fmt.Errorf("Username mismatch. %s != %s", a.Username, b.Username)
	}
	if a.Name != b.Name {
		return fmt.Errorf("Name mismatch. %s != %s", a.Name, b.Name)
	}
	if a.Path != b.Path {
		return fmt.Errorf("Path mismatch. %s != %s", a.Path, b.Path)
	}
	return nil
}

func TestGetFromCollection(t *testing.T) {
	tr := newTestRunner(t)
	defer tr.Delete()

	// Save a dataset with a body
	_, err := tr.SaveWithParams(&SaveParams{
		Ref:      "me/cities_ds",
		BodyPath: "testdata/cities_2/body.csv",
	})
	if err != nil {
		t.Fatal(err)
	}

	// get from the repo
	ds := tr.MustGet(t, "me/cities_ds")
	expect := dsref.ConvertDatasetToVersionInfo(ds)
	pro, err := tr.Instance.activeProfile(tr.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	expect.ProfileID = pro.ID.Encode()
	expect.CommitCount = 1

	// fetch from the collection
	got, err := tr.Instance.Collection().Get(tr.Ctx, &CollectionGetParams{Ref: "me/cities_ds"})
	if err != nil {
		t.Fatalf("error getting from collection by ref: %s", err)
	}

	if diff := cmp.Diff(expect, *got); diff != "" {
		t.Errorf("get from collection mistmatch (-want +got):\n%s", diff)
	}

	got, err = tr.Instance.Collection().Get(tr.Ctx, &CollectionGetParams{InitID: expect.InitID})
	if err != nil {
		t.Fatalf("error getting from collection by initID: %s", err)
	}

	if diff := cmp.Diff(expect, *got); diff != "" {
		t.Errorf("get from collection mistmatch (-want +got):\n%s", diff)
	}
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

			inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)
			p := &CollectionListParams{List: params.List{Limit: 30, Offset: 0}}
			res, _, err := inst.Collection().List(ctx, p)
			if err != nil {
				t.Errorf("error listing dataset: %s", err.Error())
			}
			// Get number from end of peername, use that to find dataset name.
			profile := node.Repo.Profiles().Owner(ctx)
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
	node, err := p2p.NewQriNode(mr, testcfg.DefaultP2PForTesting(), event.NilBus, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	inst := NewInstanceFromConfigAndNode(ctx, testcfg.DefaultConfigForTesting(), node)

	text, err := inst.Collection().ListRawRefs(ctx, &EmptyParams{})
	if err != nil {
		t.Fatal(err)
	}

	expect := dstest.Template(t, `0 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      cities
  Path:      {{ .citiesPath }}
  Published: false
1 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      counter
  Path:      {{ .counterPath }}
  Published: false
2 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      craigslist
  Path:      {{ .craigslistPath }}
  Published: false
3 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      movies
  Path:      {{ .moviesPath }}
  Published: false
4 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      sitemap
  Path:      {{ .sitemapPath }}
  Published: false
`, map[string]string{
		"ProfileID":      "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		"citiesPath":     "/mem/QmPWCzaxFoxAu5wS8qXkL6tSA7aR2Lpcwykfz1TbhhpuDp",
		"counterPath":    "/mem/QmVN68yJdLCstVj7YiDjoDvbuxnWKL57D5EAszM7SxtXi3",
		"craigslistPath": "/mem/QmTzSsKodVuQRBbcAnYhh8iHSnCA59CNsJzJxue9if9yXN",
		"moviesPath":     "/mem/QmXkLt1xHqtJjjGoT2reGZLBFELsioWkJ24yDjchGpu63W",
		"sitemapPath":    "/mem/QmdotsdAr5w32jToY13q4VR9CYdN9hTpkivJjwRELhGkxa",
	})

	if diff := cmp.Diff(expect, text); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
