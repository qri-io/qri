package actions

import (
	"context"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestUpdateRemoteDataset(t *testing.T) {
	// TODO (b5) - restore
}

func TestAddDataset(t *testing.T) {
	ctx := context.Background()
	node := newTestNode(t)

	if err := AddDataset(ctx, node, nil, "", &repo.DatasetRef{Peername: "foo", Name: "bar"}); err == nil {
		t.Error("expected add of invalid ref to error")
	}
	// Create test nodes.
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
	if err := AddDataset(ctx, peers[0], nil, "", &repo.DatasetRef{Peername: p2Pro.Peername, Name: "cities"}); err != nil {
		t.Error(err.Error())
	}
}

func TestDataset(t *testing.T) {
	rmf := func(t *testing.T) repo.Repo {
		store := cafs.NewMapstore()
		testPeerProfile.PrivKey = privKey
		mr, err := repo.NewMemRepo(testPeerProfile, store, qfs.NewMemFS(), profile.NewMemStore())
		if err != nil {
			panic(err)
		}
		return mr
	}
	DatasetTests(t, rmf)
}

func TestSaveDataset(t *testing.T) {
	ctx := context.Background()
	n := newTestNode(t)

	// test Dry run
	ds := &dataset.Dataset{
		Name:      "dry_run_test",
		Structure: &dataset.Structure{Format: "json", Schema: map[string]interface{}{"type": "array"}},
		Meta: &dataset.Meta{
			Title: "test title",
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	ref, err := SaveDataset(ctx, n.Repo, devNull, ds, nil, nil, SaveDatasetSwitches{DryRun: true, ShouldRender: true})
	if err != nil {
		t.Errorf("dry run error: %s", err.Error())
	}
	if ref.AliasString() != "peer/dry_run_test" {
		t.Errorf("ref alias mismatch. expected: '%s' got: '%s'", "peer/dry_run_test", ref.AliasString())
	}

	ds = &dataset.Dataset{
		Peername: ref.Peername,
		Name:     "test_save",
		Commit: &dataset.Commit{
			Title:   "initial commit",
			Message: "manually create a baseline dataset",
		},
		Meta: &dataset.Meta{
			Title: "another test dataset",
		},
		Structure: &dataset.Structure{Format: "json", Schema: map[string]interface{}{"type": "array"}},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	// test save
	ref, err = SaveDataset(ctx, n.Repo, devNull, ds, nil, nil, SaveDatasetSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Error(err)
	}
	secrets := map[string]string{
		"bar": "secret",
	}

	ds = &dataset.Dataset{
		Peername: ref.Peername,
		Name:     ref.Name,
		Commit: &dataset.Commit{
			Title:   "add transform script",
			Message: "adding an append-only transform script",
		},
		Transform: &dataset.Transform{
			Syntax: "starlark",
			Config: map[string]interface{}{
				"foo": "config",
			},
			ScriptBytes: []byte(`def transform(ds,ctx): 
  ctx.get_config("foo")
  ctx.get_secret("bar")
  ds.set_body(["hey"])`),
		},
	}
	ds.Transform.OpenScriptFile(ctx, nil)

	// dryrun should work
	ref, err = SaveDataset(ctx, n.Repo, devNull, ds, secrets, nil, SaveDatasetSwitches{DryRun: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err)
	}

	ds = &dataset.Dataset{
		Peername: ref.Peername,
		Name:     ref.Name,
		Commit: &dataset.Commit{
			Title:   "add transform script",
			Message: "adding an append-only transform script",
		},
		Transform: &dataset.Transform{
			Syntax: "starlark",
			Config: map[string]interface{}{
				"foo": "config",
			},
			ScriptBytes: []byte(`def transform(ds,ctx): 
  ctx.get_config("foo")
  ctx.get_secret("bar")
  ds.set_body(["hey"])`),
		},
	}
	ds.Transform.OpenScriptFile(ctx, nil)

	// test save with transform
	ref, err = SaveDataset(ctx, n.Repo, devNull, ds, secrets, nil, SaveDatasetSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err)
	}

	// save new manual changes
	ds = &dataset.Dataset{
		Peername: ref.Peername,
		Name:     ref.Name,
		Commit: &dataset.Commit{
			Title:   "update meta",
			Message: "manual change that'll negate previous transform",
		},
		Meta: &dataset.Meta{
			Title:       "updated title",
			Description: "updated description",
		},
	}

	ref, err = SaveDataset(ctx, n.Repo, devNull, ds, nil, nil, SaveDatasetSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Error(err)
	}

	if ref.Dataset.Transform != nil {
		t.Error("expected manual save to remove transform")
	}

	// recall previous transform
	tfds, err := base.Recall(ctx, n.Repo, "tf", ref)
	if err != nil {
		t.Error(err)
	}

	ds = &dataset.Dataset{
		Peername: ref.Peername,
		Name:     ref.Name,
		Commit: &dataset.Commit{
			Title:   "re-run transform",
			Message: "recall transform & re-run it",
		},
		Transform: tfds.Transform,
	}
	if err := ds.Transform.OpenScriptFile(ctx, n.Repo.Filesystem()); err != nil {
		t.Error(err)
	}

	ref, err = SaveDataset(ctx, n.Repo, devNull, ds, secrets, nil, SaveDatasetSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Error(err)
	}
	if ref.Dataset.Transform == nil {
		t.Error("expected recalled transform to be present")
	}
}

func TestSaveDatasetWithoutStructureOrBody(t *testing.T) {
	ctx := context.Background()
	n := newTestNode(t)

	ds := &dataset.Dataset{
		Name: "no_st_or_body_test",
		Meta: &dataset.Meta{
			Title: "test title",
		},
	}

	_, err := SaveDataset(ctx, n.Repo, devNull, ds, nil, nil, SaveDatasetSwitches{ShouldRender: true})
	expect := "creating a new dataset requires a structure or a body"
	if err == nil || err.Error() != expect {
		t.Errorf("expected error, but got %s", err.Error())
	}
}

func TestSaveDatasetReplace(t *testing.T) {
	ctx := context.Background()
	n := newTestNode(t)

	ds := &dataset.Dataset{
		Peername: "me",
		Name:     "test_save",
		Meta: &dataset.Meta{
			Title: "another test dataset",
		},
		Structure: &dataset.Structure{Format: "json", Schema: map[string]interface{}{"type": "array"}},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	// test save
	_, err := SaveDataset(ctx, n.Repo, devNull, ds, nil, nil, SaveDatasetSwitches{Pin: true})
	if err != nil {
		t.Error(err)
	}

	ds = &dataset.Dataset{
		Peername:  "me",
		Name:      "test_save",
		Structure: &dataset.Structure{Format: "json", Schema: map[string]interface{}{"type": "object"}},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte(`{"foo":"bar"}`)))

	ref, err := SaveDataset(ctx, n.Repo, devNull, ds, nil, nil, SaveDatasetSwitches{Replace: true, Pin: true})
	if err != nil {
		t.Error(err)
	}

	if err := base.ReadDataset(ctx, n.Repo, &ref); err != nil {
		t.Error(err)
	}

	if ref.Dataset.Meta != nil {
		t.Error("expected overwritten meta to be nil")
	}
}

type RepoMakerFunc func(t *testing.T) repo.Repo
type RepoTestFunc func(t *testing.T, rmf RepoMakerFunc)

func DatasetTests(t *testing.T, rmf RepoMakerFunc) {
	for _, test := range []RepoTestFunc{
		testSaveDataset,
		testReadDataset,
		testRenameDataset,
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

	ref := addCitiesDataset(t, n.Repo)
	return n, ref
}

func testReadDataset(t *testing.T, rmf RepoMakerFunc) {
	ctx := context.Background()
	n, ref := createDataset(t, rmf)

	if err := base.ReadDataset(ctx, n.Repo, &ref); err != nil {
		t.Error(err.Error())
		return
	}

	if ref.Dataset == nil {
		t.Error("expected dataset to not equal nil")
		return
	}
}

func testRenameDataset(t *testing.T, rmf RepoMakerFunc) {
	ctx := context.Background()
	node, ref := createDataset(t, rmf)

	b := &repo.DatasetRef{
		Name:     "cities2",
		Peername: "me",
	}

	if err := base.ModifyDatasetRef(ctx, node.Repo, &ref, b, true); err != nil {
		t.Error(err)
		return
	}

	if err := base.ReadDataset(ctx, node.Repo, b); err != nil {
		t.Error(err)
		return
	}

	if b.Dataset == nil {
		t.Error("expected dataset to not equal nil")
		return
	}
}

func testDeleteDataset(t *testing.T, rmf RepoMakerFunc) {
	ctx := context.Background()
	node, ref := createDataset(t, rmf)

	if err := base.DeleteDataset(ctx, node.Repo, &ref); err != nil {
		t.Error(err.Error())
		return
	}
}
