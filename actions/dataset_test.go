package actions

import (
	"context"
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regserver/mock"
)

func TestUpdateDataset(t *testing.T) {
	node := newTestNode(t)
	cities := addCitiesDataset(t, node)

	expect := "transform script is required to automate updates to your own datasets"
	if _, _, err := UpdateDataset(node, &cities, false, true); err == nil {
		t.Error("expected update without transform to error")
	} else if err.Error() != expect {
		t.Errorf("error mismatch. %s != %s", expect, err.Error())
	}

	now := addNowTransformDataset(t, node)
	now, _, err := UpdateDataset(node, &now, false, false)
	if err != nil {
		t.Error(err)
	}
}

func TestAddDataset(t *testing.T) {
	node := newTestNode(t)

	if err := AddDataset(node, &repo.DatasetRef{Peername: "foo", Name: "bar"}); err == nil {
		t.Error("expected add of invalid ref to error")
	}

	// Create test nodes.
	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 2)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
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
	p2Pro, _ := peers[1].Repo.Profile()
	if err := AddDataset(peers[0], &repo.DatasetRef{Peername: p2Pro.Peername, Name: "cities"}); err != nil {
		t.Error(err.Error())
	}
}

func TestDataset(t *testing.T) {
	rc, _ := mock.NewMockServer()

	rmf := func(t *testing.T) repo.Repo {
		mr, err := repo.NewMemRepo(testPeerProfile, cafs.NewMapstore(), profile.NewMemStore(), rc)
		if err != nil {
			panic(err)
		}
		// mr.SetPrivateKey(privKey)
		return mr
	}
	DatasetTests(t, rmf)
}

func TestSaveDataset(t *testing.T) {
	n := newTestNode(t)

	// test Dry run
	ds := &dataset.DatasetPod{
		Name:      "dry_run_test",
		Structure: &dataset.StructurePod{Format: dataset.JSONDataFormat.String(), Schema: map[string]interface{}{"type": "array"}},
		Meta: &dataset.Meta{
			Title: "test title",
		},
		BodyBytes: []byte("[]"),
	}

	ref, _, err := SaveDataset(n, ds, true, false)
	if err != nil {
		t.Errorf("dry run error: %s", err.Error())
	}
	if ref.AliasString() != "peer/dry_run_test" {
		t.Errorf("ref alias mismatch. expected: '%s' got: '%s'", "peer/dry_run_test", ref.AliasString())
	}
}

type RepoMakerFunc func(t *testing.T) repo.Repo
type RepoTestFunc func(t *testing.T, rmf RepoMakerFunc)

func DatasetTests(t *testing.T, rmf RepoMakerFunc) {
	for _, test := range []RepoTestFunc{
		testSaveDataset,
		testReadDataset,
		testRenameDataset,
		testDeleteDataset,
		testEventsLog,
	} {
		test(t, rmf)
	}
}

func testSaveDataset(t *testing.T, rmf RepoMakerFunc) {
	createDataset(t, rmf)
}

func createDataset(t *testing.T, rmf RepoMakerFunc) (*p2p.QriNode, repo.DatasetRef) {
	r := rmf(t)
	r.SetProfile(testPeerProfile)
	n, err := p2p.NewQriNode(r, config.DefaultP2PForTesting())
	if err != nil {
		t.Error(err.Error())
		return n, repo.DatasetRef{}
	}

	ref := addCitiesDataset(t, n)
	return n, ref
}

func testReadDataset(t *testing.T, rmf RepoMakerFunc) {
	n, ref := createDataset(t, rmf)

	if err := base.ReadDataset(n.Repo, &ref); err != nil {
		t.Error(err.Error())
		return
	}

	if ref.Dataset == nil {
		t.Error("expected dataset to not equal nil")
		return
	}
}

func testRenameDataset(t *testing.T, rmf RepoMakerFunc) {
	node, ref := createDataset(t, rmf)

	b := &repo.DatasetRef{
		Name:     "cities2",
		Peername: "me",
	}

	if err := RenameDataset(node, &ref, b); err != nil {
		t.Error(err.Error())
		return
	}

	if err := base.ReadDataset(node.Repo, b); err != nil {
		t.Error(err.Error())
		return
	}

	if b.Dataset == nil {
		t.Error("expected dataset to not equal nil")
		return
	}
}

func testDeleteDataset(t *testing.T, rmf RepoMakerFunc) {
	node, ref := createDataset(t, rmf)

	if err := DeleteDataset(node, &ref); err != nil {
		t.Error(err.Error())
		return
	}
}

func testEventsLog(t *testing.T, rmf RepoMakerFunc) {
	node, ref := createDataset(t, rmf)
	pinner := true

	b := &repo.DatasetRef{
		Name:      "cities2",
		ProfileID: ref.ProfileID,
	}

	if err := RenameDataset(node, &ref, b); err != nil {
		t.Error(err.Error())
		return
	}

	if err := base.PinDataset(node.Repo, *b); err != nil {
		if err == repo.ErrNotPinner {
			pinner = false
		} else {
			t.Error(err.Error())
			return
		}
	}

	// TODO - calling unpin followed by delete will trigger two unpin events,
	// which based on our current architecture can and will probably cause problems
	// we should either hardern every unpin implementation to not error on multiple
	// calls to unpin the same hash, or include checks in the delete method
	// and only call unpin if the hash is in fact pinned
	// if err := act.UnpinDataset(b); err != nil && err != repo.ErrNotPinner {
	// 	t.Error(err.Error())
	// 	return
	// }

	if err := DeleteDataset(node, b); err != nil {
		t.Error(err.Error())
		return
	}

	events, err := node.Repo.Events(10, 0)
	if err != nil {
		t.Error(err.Error())
		return
	}

	ets := []repo.EventType{repo.ETDsDeleted, repo.ETDsUnpinned, repo.ETDsPinned, repo.ETDsRenamed, repo.ETDsPinned, repo.ETDsCreated}

	if !pinner {
		ets = []repo.EventType{repo.ETDsDeleted, repo.ETDsRenamed, repo.ETDsCreated}
	}

	if len(events) != len(ets) {
		t.Errorf("event log length mismatch. expected: %d, got: %d", len(ets), len(events))
		t.Log("event log:")
		for i, e := range events {
			t.Logf("\t%d: %s", i, e.Type)
		}
		return
	}

	for i, et := range ets {
		if events[i].Type != et {
			t.Errorf("case %d eventType mismatch. expected: %s, got: %s", i, et, events[i].Type)
		}
	}
}
