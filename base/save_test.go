package base

import (
	"context"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestSaveDataset(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)

	// test Dry run
	ds := &dataset.Dataset{
		Name:      "dry_run_test",
		Structure: &dataset.Structure{Format: "json", Schema: map[string]interface{}{"type": "array"}},
		Meta: &dataset.Meta{
			Title: "test title",
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	ref, err := SaveDataset(ctx, r, devNull, ds, nil, nil, SaveDatasetSwitches{DryRun: true, ShouldRender: true})
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
	ref, err = SaveDataset(ctx, r, devNull, ds, nil, nil, SaveDatasetSwitches{Pin: true, ShouldRender: true})
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
	ref, err = SaveDataset(ctx, r, devNull, ds, secrets, nil, SaveDatasetSwitches{DryRun: true, ShouldRender: true})
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
	ref, err = SaveDataset(ctx, r, devNull, ds, secrets, nil, SaveDatasetSwitches{Pin: true, ShouldRender: true})
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

	ref, err = SaveDataset(ctx, r, devNull, ds, nil, nil, SaveDatasetSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Error(err)
	}

	if ref.Dataset.Transform != nil {
		t.Error("expected manual save to remove transform")
	}

	// recall previous transform
	tfds, err := Recall(ctx, r, "tf", ref)
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
	if err := ds.Transform.OpenScriptFile(ctx, r.Filesystem()); err != nil {
		t.Error(err)
	}

	ref, err = SaveDataset(ctx, r, devNull, ds, secrets, nil, SaveDatasetSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Error(err)
	}
	if ref.Dataset == nil {
		t.Error("expected dataset pointer to exist")
	} else if ref.Dataset.Transform == nil {
		t.Error("expected recalled transform to be present")
	}
}

func TestSaveDatasetWithoutStructureOrBody(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)

	ds := &dataset.Dataset{
		Name: "no_st_or_body_test",
		Meta: &dataset.Meta{
			Title: "test title",
		},
	}

	_, err := SaveDataset(ctx, r, devNull, ds, nil, nil, SaveDatasetSwitches{ShouldRender: true})
	expect := "creating a new dataset requires a structure or a body"
	if err == nil || err.Error() != expect {
		t.Errorf("expected error, but got %s", err.Error())
	}
}

func TestSaveDatasetReplace(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)

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
	_, err := SaveDataset(ctx, r, devNull, ds, nil, nil, SaveDatasetSwitches{Pin: true})
	if err != nil {
		t.Error(err)
	}

	ds = &dataset.Dataset{
		Peername:  "me",
		Name:      "test_save",
		Structure: &dataset.Structure{Format: "json", Schema: map[string]interface{}{"type": "object"}},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte(`{"foo":"bar"}`)))

	ref, err := SaveDataset(ctx, r, devNull, ds, nil, nil, SaveDatasetSwitches{Replace: true, Pin: true})
	if err != nil {
		t.Error(err)
	}

	if err := ReadDataset(ctx, r, &ref); err != nil {
		t.Error(err)
	}

	if ref.Dataset.Meta != nil {
		t.Error("expected overwritten meta to be nil")
	}
}

func TestCreateDataset(t *testing.T) {
	ctx := context.Background()
	streams := ioes.NewDiscardIOStreams()
	store := cafs.NewMapstore()
	r, err := repo.NewMemRepo(testPeerProfile, store, qfs.NewMemFS(), profile.NewMemStore())
	if err != nil {
		t.Fatal(err.Error())
	}

	ds := &dataset.Dataset{
		Name:   "foo",
		Meta:   &dataset.Meta{Title: "test"},
		Commit: &dataset.Commit{Title: "hello"},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	if _, err := CreateDataset(ctx, r, streams, &dataset.Dataset{}, &dataset.Dataset{}, false, true, false, true); err == nil {
		t.Error("expected bad dataset to error")
	}

	ref, err := CreateDataset(ctx, r, streams, ds, &dataset.Dataset{}, false, true, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	refs, err := r.References(0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Errorf("ref length mismatch. expected 1, got: %d", len(refs))
	}

	ds.Meta.Title = "an update"
	ds.PreviousPath = ref.Path
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	prev := ref.Dataset

	ref, err = CreateDataset(ctx, r, streams, ds, prev, false, true, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	refs, err = r.References(0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Errorf("ref length mismatch. expected 1, got: %d", len(refs))
	}

	ds.PreviousPath = ref.Path
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))
	prev = ref.Dataset

	if ref, err = CreateDataset(ctx, r, streams, ds, prev, false, true, false, true); err == nil {
		t.Error("expected unchanged dataset with no force flag to error")
	}

	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))
	if ref, err = CreateDataset(ctx, r, streams, ds, prev, false, true, true, true); err != nil {
		t.Errorf("unexpected force-save error: %s", err)
	}
}
