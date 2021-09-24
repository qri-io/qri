package base

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
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

	if _, err := CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), r.Profiles().Owner(ctx), &dataset.Dataset{}, &dataset.Dataset{}, SaveSwitches{Pin: true, ShouldRender: true}); err == nil {
		t.Error("expected bad dataset to error")
	}

	createdDs, err := CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), r.Profiles().Owner(ctx), ds, &dataset.Dataset{}, SaveSwitches{Pin: true, ShouldRender: true})
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

	createdDs, err = CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), r.Profiles().Owner(ctx), ds, createdDs, SaveSwitches{Pin: true, ShouldRender: true})
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
	prevPath := createdDs.Path

	// Need to reset because CreateDataset clears the name before writing to ipfs. Remove the
	// reliance on CreateDataset needing the ds.Name field.
	ds.Name = dsName
	ds.PreviousPath = prevPath
	ds.SetBodyFile(qfs.NewMemfileBytes("/body.json", []byte("[]")))

	if createdDs, err = CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), r.Profiles().Owner(ctx), ds, createdDs, SaveSwitches{Pin: true, ShouldRender: true}); err == nil {
		t.Error("expected unchanged dataset with no force flag to error")
	}

	ds.Name = dsName
	ds.PreviousPath = prevPath
	ds.SetBodyFile(qfs.NewMemfileBytes("/body.json", []byte("[]")))
	if createdDs, err = CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), r.Profiles().Owner(ctx), ds, createdDs, SaveSwitches{ForceIfNoChanges: true, Pin: true, ShouldRender: true}); err != nil {
		t.Errorf("unexpected force-save error: %s", err)
	}
}

func TestPrepareSaveRef(t *testing.T) {
	logbook.NewTimestamp = func() int64 { return 0 }
	defer func() {
		logbook.NewTimestamp = func() int64 { return time.Now().Unix() }
	}()

	r := newTestRepo(t)
	ctx := context.Background()

	author := r.Profiles().Owner(ctx)
	book := r.Logbook()

	book.WriteDatasetInit(ctx, author, "cities")
	book.WriteDatasetInit(ctx, author, "Bad_Case")

	bad := []struct {
		refStr, filepath string
		newName          bool
		expect           dsref.Ref
		expectIsNew      bool
		err              string
	}{
		{"me/invalid name", "", false, dsref.Ref{Username: "me", Name: "invalid"}, false, dsref.ErrDescribeValidName.Error()},
		{"me/cities", "", true, dsref.Ref{Username: "peer", Name: "cities", ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", InitID: "r7kr6djpgu2hm5fprxalfcsgehacoxomqse4c7nubu5mw6qcz57q"}, false, "name already in use"},
		{"me/cities@/ipfs/foo", "", true, dsref.Ref{Username: "me", Name: "cities", ProfileID: "", InitID: ""}, false, dsref.ErrNotHumanFriendly.Error()},
		{"alice/not_this_user", "", true, dsref.Ref{Username: "alice", Name: "not_this_user", ProfileID: "", InitID: ""}, false, "cannot save using a different username than \"peer\""},
		{"me/New_Bad_Case", "", false, dsref.Ref{Username: "peer", Name: "New_Bad_Case", InitID: ""}, true, dsref.ErrBadCaseName.Error()},
	}

	for _, c := range bad {
		t.Run(fmt.Sprintf("bad_%s", c.refStr), func(t *testing.T) {
			ref, isNew, err := PrepareSaveRef(ctx, author, book, book, c.refStr, c.filepath, c.newName)
			if !c.expect.Equals(ref) {
				t.Errorf("resulting ref mismatch. want:\n%#v\ngot:\n%#v", c.expect, ref)
			}
			if c.expectIsNew != isNew {
				t.Errorf("isNew result mismatch. want %t got %t", c.expectIsNew, isNew)
			}
			if err == nil {
				t.Fatal("expected error, got none")
			}
			if c.err != err.Error() {
				t.Errorf("error mismatch. want %q got %q", c.err, err.Error())
			}
		})
	}

	good := []struct {
		refStr, filepath string
		newName          bool
		expect           dsref.Ref
		expectIsNew      bool
	}{
		{"", "", false, dsref.Ref{Username: "peer", Name: "dataset", InitID: "2fxdc6hvi5gdraujcru5vnaluuuf57x345eirtwwtwitmjhr54ca"}, true},
		{"me/cities", "", false, dsref.Ref{Username: "peer", Name: "cities", ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", InitID: "r7kr6djpgu2hm5fprxalfcsgehacoxomqse4c7nubu5mw6qcz57q"}, false},
		{"", "/path/to/data/apples.csv", false, dsref.Ref{Username: "peer", Name: "apples", InitID: "bj2srktro6zxsvork6stjzecq4f4kaii2xg2q2n6b4gwk2robf2q"}, true},
		{"", "/path/to/data/apples.csv", true, dsref.Ref{Username: "peer", Name: "apples_2", InitID: "tbrfupxauhuc6rwamyejr35w4nw2icgcxvm4f6fnftyaoyeo7ida"}, true},
		{"me/Bad_Case", "", false, dsref.Ref{Username: "peer", Name: "Bad_Case", ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", InitID: "setbycsqt5gwyg3fmcm4ty37dzk5ohhq4oxk2hif64fkdhi6naca"}, false},
	}

	for _, c := range good {
		t.Run(fmt.Sprintf("good_%s", c.refStr), func(t *testing.T) {
			ref, isNew, err := PrepareSaveRef(ctx, author, book, book, c.refStr, c.filepath, c.newName)
			if err != nil {
				t.Fatalf("unexpected error: %q", err)
			}
			if !c.expect.Equals(ref) {
				t.Errorf("resulting ref mismatch. want:\n%#v\ngot:\n%#v", c.expect, ref)
			}
			if c.expectIsNew != isNew {
				t.Errorf("isNew result mismatch. want %t got %t", c.expectIsNew, isNew)
			}
		})
	}

}

func TestInferValues(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	pro := r.Profiles().Owner(ctx)
	ds := &dataset.Dataset{}
	if err := InferValues(pro, ds); err != nil {
		t.Error(err)
	}
	expectAuthorID := `QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt`
	if diff := cmp.Diff(expectAuthorID, ds.Commit.Author.ID); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestInferValuesDontOverwriteSchema(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	pro := r.Profiles().Owner(ctx)

	ds := &dataset.Dataset{
		Name: "animals",
		Structure: &dataset.Structure{
			Format: "csv",
			Schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "array",
					"items": []interface{}{
						map[string]interface{}{"title": "animal", "type": "number"},
						map[string]interface{}{"title": "noise", "type": "number"},
						map[string]interface{}{"title": "height", "type": "number"},
					},
				},
			},
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("animals.csv",
		[]byte("Animal,Sound,Weight\ncat,meow,1.4\ndog,bark,3.7\n")))
	if err := InferValues(pro, ds); err != nil {
		t.Error(err)
	}

	if ds.Structure.Format != "csv" {
		t.Errorf("expected format CSV, got %s", ds.Structure.Format)
	}
	if ds.Structure.FormatConfig == nil {
		t.Errorf("expected format config to be non-nil")
	}

	actual := datasetSchemaToJSON(ds)
	expect := `{"items":{"items":[{"title":"animal","type":"number"},{"title":"noise","type":"number"},{"title":"height","type":"number"}],"type":"array"},"type":"array"}`

	if expect != actual {
		t.Errorf("mismatched schema, expected %q, got %q", expect, actual)
	}
}

func TestMaybeAddDefaultViz(t *testing.T) {
	ds := &dataset.Dataset{
		Name: "animals",
		Structure: &dataset.Structure{
			Format: "csv",
		},
	}
	MaybeAddDefaultViz(ds)
	if ds.Viz == nil {
		t.Fatal("expected MaybeAddDefaultViz to create a viz component")
	}
	if ds.Viz.Format != "html" {
		t.Errorf("expected default viz format to equal 'html'. got: %s", ds.Viz.Format)
	}
	if ds.Viz.ScriptFile().FileName() != "viz.html" {
		t.Errorf("expected default viz file to equal 'viz.html'. got: %s", ds.Viz.ScriptFile().FileName())
	}
}

func datasetSchemaToJSON(ds *dataset.Dataset) string {
	js, err := json.Marshal(ds.Structure.Schema)
	if err != nil {
		return err.Error()
	}
	return string(js)
}
