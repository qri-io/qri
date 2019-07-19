package actions

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regserver/mock"
)

func TestUpdateRemoteDataset(t *testing.T) {
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

	base.ReadDataset(peers[0].Repo, &now)

	ds := &dataset.Dataset{
		Peername: now.Peername,
		Name:     now.Name,
		Commit: &dataset.Commit{
			Title:   "total overwrite",
			Message: "manually create a silly change",
		},
		Meta: &dataset.Meta{
			Title: "another test dataset",
		},
		Structure: &dataset.Structure{Format: "json", Schema: map[string]interface{}{"type": "array"}},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	// run a local update to advance history
	now0, err := SaveDataset(peers[0], ds, nil, nil, SaveDatasetSwitches{ Pin: true, ShouldRender: true })
	if err != nil {
		t.Error(err)
	}

	now1, err := UpdateRemoteDataset(peers[1], &now, false)
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
		store := cafs.NewMapstore()
		mr, err := repo.NewMemRepo(testPeerProfile, store, qfs.NewMemFS(store), profile.NewMemStore(), rc)
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
	ds := &dataset.Dataset{
		Name:      "dry_run_test",
		Structure: &dataset.Structure{Format: "json", Schema: map[string]interface{}{"type": "array"}},
		Meta: &dataset.Meta{
			Title: "test title",
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	ref, err := SaveDataset(n, ds, nil, nil, SaveDatasetSwitches{ DryRun: true, ShouldRender: true })
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
	ref, err = SaveDataset(n, ds, nil, nil, SaveDatasetSwitches{ Pin: true, ShouldRender: true })
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
	ds.Transform.OpenScriptFile(nil)

	// dryrun should work
	ref, err = SaveDataset(n, ds, secrets, nil, SaveDatasetSwitches{ DryRun: true, ShouldRender: true })
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
	ds.Transform.OpenScriptFile(nil)

	// test save with transform
	ref, err = SaveDataset(n, ds, secrets, nil, SaveDatasetSwitches{ Pin: true, ShouldRender: true })
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

	ref, err = SaveDataset(n, ds, nil, nil, SaveDatasetSwitches{ Pin: true, ShouldRender: true })
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

	ds = &dataset.Dataset{
		Peername: ref.Peername,
		Name:     ref.Name,
		Commit: &dataset.Commit{
			Title:   "re-run transform",
			Message: "recall transform & re-run it",
		},
		Transform: tfds.Transform,
	}
	if err := ds.Transform.OpenScriptFile(n.Repo.Filesystem()); err != nil {
		t.Error(err)
	}

	ref, err = SaveDataset(n, ds, secrets, nil, SaveDatasetSwitches{ Pin: true, ShouldRender: true })
	if err != nil {
		t.Error(err)
	}
	if ref.Dataset.Transform == nil {
		t.Error("expected recalled transform to be present")
	}
}

func TestSaveDatasetWithoutStructureOrBody(t *testing.T) {
	n := newTestNode(t)

	ds := &dataset.Dataset{
		Name: "no_st_or_body_test",
		Meta: &dataset.Meta{
			Title: "test title",
		},
	}

	_, err := SaveDataset(n, ds, nil, nil, SaveDatasetSwitches{ ShouldRender: true })
	expect := "creating a new dataset requires a structure or a body"
	if err == nil || err.Error() != expect {
		t.Errorf("expected error, but got %s", err.Error())
	}
}

func TestSaveDatasetReplace(t *testing.T) {
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
	_, err := SaveDataset(n, ds, nil, nil, SaveDatasetSwitches{ Pin: true })
	if err != nil {
		t.Error(err)
	}

	ds = &dataset.Dataset{
		Peername: "me",
		Name:     "test_save",
		Structure: &dataset.Structure{Format: "json", Schema: map[string]interface{}{"type": "object"}},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte(`{"foo":"bar"}`)))

	ref, err := SaveDataset(n, ds, nil, nil, SaveDatasetSwitches{ Replace: true, Pin: true })
	if err != nil {
		t.Error(err)
	}

	if err := base.ReadDataset(n.Repo, &ref); err != nil {
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

func TestReplaceRefIfMoreRecent(t *testing.T) {
	node := newTestNode(t)
	older := time.Date(2019, 1, 1, 12, 0, 0, 0, time.UTC)
	newer := older.AddDate(1, 0, 0)
	cases := []struct {
		description string
		a, b        repo.DatasetRef
		path        string
	}{
		{
			"first dataset is older then the second",
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_older",
				Path:      "/map/first",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: older,
					},
				},
			},
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_older",
				Path:      "/map/second",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			"/map/second",
		},
		{
			"first dataset is newer then the second",
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_newer",
				Path:      "/map/first",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_newer",
				Path:      "/map/second",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: older,
					},
				},
			},
			"/map/first",
		},
		{
			"first dataset is same time as the the second",
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_same",
				Path:      "/map/first",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_same",
				Path:      "/map/second",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			"/map/second",
		},
	}

	for _, c := range cases {
		if err := node.Repo.PutRef(c.a); err != nil {
			t.Fatal(err)
		}
		if err := ReplaceRefIfMoreRecent(node, &c.a, &c.b); err != nil {
			t.Fatal(err)
		}
		ref, err := node.Repo.GetRef(repo.DatasetRef{Peername: c.a.Peername, Name: c.a.Name})
		if err != nil {
			t.Fatal(err)
		}
		if ref.Path != c.path {
			t.Errorf("case '%s', ref path error, expected: '%s', got: '%s'", c.description, c.path, ref.Path)
		}
	}

	casesError := []struct {
		description string
		a, b        repo.DatasetRef
		err         string
	}{
		{
			"original ref has no timestamp & should error",
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "err",
				Path:      "/map/first",
				ProfileID: "id",
			},
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "err",
				Path:      "/map/second",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			"previous dataset ref is not fully derefernced",
		},
		{
			"added ref has no timestamp & should error",
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "err",
				Path:      "/map/first",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			repo.DatasetRef{},
			"added dataset ref is not fully dereferenced",
		},
	}

	for _, c := range casesError {
		if err := node.Repo.PutRef(c.a); err != nil {
			t.Fatal(err)
		}
		err := ReplaceRefIfMoreRecent(node, &c.a, &c.b)
		if err == nil {
			t.Errorf("case '%s' did not error", c.description)
		}
		if err.Error() != c.err {
			t.Errorf("case '%s', error mismatch. expected: '%s', got: '%s'", c.description, c.err, err.Error())
		}
	}
}
