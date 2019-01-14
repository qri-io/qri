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

func TestUpdateDatasetLocal(t *testing.T) {
	node := newTestNode(t)
	cities := addCitiesDataset(t, node)

	expect := "transform script is required to automate updates to your own datasets"
	if _, _, err := UpdateDataset(node, &cities, nil, nil, false, true); err == nil {
		t.Error("expected update without transform to error")
	} else if err.Error() != expect {
		t.Errorf("error mismatch. %s != %s", expect, err.Error())
	}

	now := addNowTransformDataset(t, node)
	prevPath := now.Path
	now, _, err := UpdateDataset(node, &now, nil, nil, false, false)
	if err != nil {
		t.Error(err)
	}

	if now.Dataset.PreviousPath != prevPath {
		t.Errorf("PreviousPath mismatch. expected: %s, got: %s", prevPath, now.Dataset.PreviousPath)
	}
}

func TestUpdateDatasetRemote(t *testing.T) {
	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 2)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	peers := asQriNodes(testPeers)
	connectMapStores(peers)

	now := addNowTransformDataset(t, peers[0])
	if err := AddDataset(peers[1], &repo.DatasetRef{Peername: now.Peername, Name: now.Name}); err != nil {
		t.Error(err)
	}

	// run a local update to advance history
	now0, _, err := UpdateDataset(peers[0], &now, nil, nil, false, false)
	if err != nil {
		t.Error(err)
	}

	now1, _, err := UpdateDataset(peers[1], &now, nil, nil, false, false)
	if err != nil {
		t.Error(err)
	}
	if !now0.Equal(now1) {
		t.Errorf("refs unequal: %s != %s", now0, now1)
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

	peers := asQriNodes(testPeers)

	connectMapStores(peers)
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

	ref, _, err := SaveDataset(n, ds, nil, nil, true, false, false)
	if err != nil {
		t.Errorf("dry run error: %s", err.Error())
	}
	if ref.AliasString() != "peer/dry_run_test" {
		t.Errorf("ref alias mismatch. expected: '%s' got: '%s'", "peer/dry_run_test", ref.AliasString())
	}

	ds = &dataset.DatasetPod{
		Peername: ref.Peername,
		Name:     "test_save",
		Commit: &dataset.CommitPod{
			Title:   "initial commit",
			Message: "manually create a baseline dataset",
		},
		Meta: &dataset.Meta{
			Title: "another test dataset",
		},
		Structure: &dataset.StructurePod{Format: dataset.JSONDataFormat.String(), Schema: map[string]interface{}{"type": "array"}},
		BodyBytes: []byte("[]"),
	}
	// test save
	ref, _, err = SaveDataset(n, ds, nil, nil, false, true, false)
	if err != nil {
		t.Error(err)
	}
	secrets := map[string]string{
		"bar": "secret",
	}

	ds = &dataset.DatasetPod{
		Peername: ref.Peername,
		Name:     ref.Name,
		Commit: &dataset.CommitPod{
			Title:   "add transform script",
			Message: "adding an append-only transform script",
		},
		Structure: &dataset.StructurePod{Format: dataset.JSONDataFormat.String(), Schema: map[string]interface{}{"type": "array"}},
		Transform: &dataset.TransformPod{
			Syntax: "starlark",
			Config: map[string]interface{}{
				"foo": "config",
			},
			ScriptBytes: []byte(`def transform(ds,ctx): 
  ctx.get_config("foo")
  ctx.get_secret("bar")
  bd = ds.get_body()
  bd.append("hey")
  ds.set_body(bd)`),
		},
	}
	// dryrun should work
	ref, _, err = SaveDataset(n, ds, secrets, nil, true, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// test save with transform
	ref, _, err = SaveDataset(n, ds, secrets, nil, false, true, false)
	if err != nil {
		t.Fatal(err)
	}

	// save new manual changes
	ds = &dataset.DatasetPod{
		Peername: ref.Peername,
		Name:     ref.Name,
		Commit: &dataset.CommitPod{
			Title:   "update meta",
			Message: "manual change that'll negate previous transform",
		},
		Meta: &dataset.Meta{
			Title:       "updated title",
			Description: "updated description",
		},
	}

	ref, _, err = SaveDataset(n, ds, nil, nil, false, true, false)
	if err != nil {
		t.Error(err)
	}

	if ref.Dataset.Transform != nil {
		t.Error("expected manual save to remove transform")
	}

	// recall previous transform
	tfds, err := Recall(n, "tf", ref)
	if err != nil {
		t.Error(err)
	}

	ds = &dataset.DatasetPod{
		Peername: ref.Peername,
		Name:     ref.Name,
		Commit: &dataset.CommitPod{
			Title:   "re-run transform",
			Message: "recall transform & re-run it",
		},
		Transform: tfds.Transform,
	}

	ref, _, err = SaveDataset(n, ds, secrets, nil, false, true, false)
	if err != nil {
		t.Error(err)
	}
	if ref.Dataset.Transform == nil {
		t.Error("expected recalled transform to be present")
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

	if err := ModifyDataset(node, &ref, b, true); err != nil {
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

	if err := ModifyDataset(node, &ref, b, true); err != nil {
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
