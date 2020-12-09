package base

import (
	"context"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
)

func TestSaveDataset(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	// TODO(dustmop): Dry run will go away soon, once apply exists

	// test Dry run
	ds := run.BuildDataset("dry_run_test", "json")
	ds.Meta = &dataset.Meta{Title: "test title"}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	ref, err := run.SaveDatasetDryRun(ds)
	if err != nil {
		t.Error(err)
	}
	if ref.Alias() != "peer/dry_run_test" {
		t.Errorf("ref alias mismatch. expected: '%s' got: '%s'",
			"peer/dry_run_test", ref.Alias())
	}

	ds = run.BuildDataset("test_save", "json")
	ds.Commit = &dataset.Commit{Title: "initial commit", Message: "manually create a dataset"}
	ds.Meta = &dataset.Meta{Title: "another test dataset"}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	// test save
	ref, err = run.SaveDataset(ds)
	if err != nil {
		t.Error(err)
	}

	// TODO(dustmop): Add tests for `qri save --apply` once transform is removed from the save
	// path and replaced with apply.
}

func TestSaveDatasetWithoutStructureOrBody(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	ds := &dataset.Dataset{
		Name: "no_st_or_body_test",
		Meta: &dataset.Meta{
			Title: "test title",
		},
	}

	_, err := run.SaveDataset(ds)
	expect := "creating a new dataset requires a structure or a body"
	if err == nil || err.Error() != expect {
		t.Errorf("expected error, but got %s", err.Error())
	}
}

func TestSaveDatasetReplace(t *testing.T) {
	run := newTestRunner(t)
	defer run.Delete()

	ds := run.BuildDataset("test_save", "json")
	ds.Meta = &dataset.Meta{Title: "another test dataset"}
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte("[]")))

	// test save
	ref, err := run.SaveDataset(ds)
	if err != nil {
		t.Error(err)
	}

	ds = run.BuildDataset("test_save", "json")
	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte(`["foo","bar"]`)))

	ref, err = run.SaveDatasetReplace(ds)
	if err != nil {
		t.Error(err)
	}

	ds, err = ReadDataset(run.Context, run.Repo, ref.Path)
	if err != nil {
		t.Error(err)
	}

	if ds.Meta != nil {
		t.Error("expected overwritten meta to be nil")
	}
}

func TestCreateDataset(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs, err := muxfs.New(ctx, []qfs.Config{
		{Type: "mem"},
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := repo.NewMemRepoWithProfile(ctx, testPeerProfile, fs, event.NewBus(ctx))
	if err != nil {
		t.Fatal(err.Error())
	}

	dsName := "foo"
	ds := &dataset.Dataset{
		Name:   dsName,
		Meta:   &dataset.Meta{Title: "test"},
		Commit: &dataset.Commit{Title: "hello"},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("/body.json", []byte("[]")))

	if _, err := CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), &dataset.Dataset{}, &dataset.Dataset{}, SaveSwitches{Pin: true, ShouldRender: true}); err == nil {
		t.Error("expected bad dataset to error")
	}

	createdDs, err := CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), ds, &dataset.Dataset{}, SaveSwitches{Pin: true, ShouldRender: true})
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

	// Need to reset because CreateDataset clears the name before writing to ipfs. Remove the
	// reliance on CreateDataset needing the ds.Name field.
	ds.Name = dsName
	ds.Meta.Title = "an update"
	ds.PreviousPath = createdDs.Path
	ds.SetBodyFile(qfs.NewMemfileBytes("/body.json", []byte("[]")))

	createdDs, err = CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), ds, createdDs, SaveSwitches{Pin: true, ShouldRender: true})
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

	// Need to reset because CreateDataset clears the name before writing to ipfs. Remove the
	// reliance on CreateDataset needing the ds.Name field.
	ds.Name = dsName
	ds.PreviousPath = createdDs.Path
	ds.SetBodyFile(qfs.NewMemfileBytes("/body.json", []byte("[]")))

	if createdDs, err = CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), ds, createdDs, SaveSwitches{Pin: true, ShouldRender: true}); err == nil {
		t.Error("expected unchanged dataset with no force flag to error")
	}

	ds.SetBodyFile(qfs.NewMemfileBytes("/body.json", []byte("[]")))
	if createdDs, err = CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), ds, createdDs, SaveSwitches{ForceIfNoChanges: true, Pin: true, ShouldRender: true}); err != nil {
		t.Errorf("unexpected force-save error: %s", err)
	}
}
