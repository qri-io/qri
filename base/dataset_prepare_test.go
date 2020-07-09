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
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
)

func TestPrepareSaveRef(t *testing.T) {
	logbook.NewTimestamp = func() int64 { return 0 }
	defer func() {
		logbook.NewTimestamp = func() int64 { return time.Now().Unix() }
	}()

	r := newTestRepo(t)

	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	book := r.Logbook()

	book.WriteDatasetInit(ctx, "cities")
	book.WriteDatasetInit(ctx, "Bad_Case")

	bad := []struct {
		refStr, filepath string
		newName          bool
		expect           dsref.Ref
		expectIsNew      bool
		err              string
	}{
		{"me/invalid name", "", false, dsref.Ref{Username: "me", Name: "invalid"}, false, dsref.ErrDescribeValidName.Error()},
		{"me/cities", "", true, dsref.Ref{Username: "peer", Name: "cities", InitID: "r7kr6djpgu2hm5fprxalfcsgehacoxomqse4c7nubu5mw6qcz57q"}, false, "name already in use"},
		{"me/cities@/ipfs/foo", "", true, dsref.Ref{Username: "me", Name: "cities", InitID: ""}, false, dsref.ErrNotHumanFriendly.Error()},
		{"alice/not_this_user", "", true, dsref.Ref{Username: "alice", Name: "not_this_user", InitID: ""}, false, "cannot save using a different username than \"peer\""},
		{"me/New_Bad_Case", "", false, dsref.Ref{Username: "peer", Name: "New_Bad_Case", InitID: ""}, true, dsref.ErrBadCaseName.Error()},
	}

	for _, c := range bad {
		t.Run(fmt.Sprintf("bad_%s", c.refStr), func(t *testing.T) {
			ref, isNew, err := PrepareSaveRef(ctx, pro, book, book, c.refStr, c.filepath, c.newName)
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
		{"me/cities", "", false, dsref.Ref{Username: "peer", Name: "cities", InitID: "r7kr6djpgu2hm5fprxalfcsgehacoxomqse4c7nubu5mw6qcz57q"}, false},
		{"", "/path/to/data/apples.csv", false, dsref.Ref{Username: "peer", Name: "apples", InitID: "bj2srktro6zxsvork6stjzecq4f4kaii2xg2q2n6b4gwk2robf2q"}, true},
		{"", "/path/to/data/apples.csv", true, dsref.Ref{Username: "peer", Name: "apples_2", InitID: "tbrfupxauhuc6rwamyejr35w4nw2icgcxvm4f6fnftyaoyeo7ida"}, true},
		{"me/Bad_Case", "", false, dsref.Ref{Username: "peer", Name: "Bad_Case", InitID: "setbycsqt5gwyg3fmcm4ty37dzk5ohhq4oxk2hif64fkdhi6naca"}, false},
	}

	for _, c := range good {
		t.Run(fmt.Sprintf("good_%s", c.refStr), func(t *testing.T) {
			ref, isNew, err := PrepareSaveRef(ctx, pro, book, book, c.refStr, c.filepath, c.newName)
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
	r := newTestRepo(t)
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}
	ds := &dataset.Dataset{}
	if err = InferValues(pro, ds); err != nil {
		t.Error(err)
	}
	expectAuthorID := `QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt`
	if diff := cmp.Diff(expectAuthorID, ds.Commit.Author.ID); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestInferValuesStructure(t *testing.T) {
	r := newTestRepo(t)
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

	ds := &dataset.Dataset{
		Name: "animals",
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("animals.csv",
		[]byte("Animal,Sound,Weight\ncat,meow,1.4\ndog,bark,3.7\n")))

	if err = InferValues(pro, ds); err != nil {
		t.Error(err)
	}

	if ds.Structure.Format != "csv" {
		t.Errorf("expected format CSV, got %s", ds.Structure.Format)
	}
	if ds.Structure.FormatConfig["headerRow"] != true {
		t.Errorf("expected format config to set headerRow set to true")
	}

	actual := datasetSchemaToJSON(ds)
	expect := `{"items":{"items":[{"title":"animal","type":"string"},{"title":"sound","type":"string"},{"title":"weight","type":"number"}],"type":"array"},"type":"array"}`

	if expect != actual {
		t.Errorf("mismatched schema, expected \"%s\", got \"%s\"", expect, actual)
	}
}

func TestInferValuesSchema(t *testing.T) {
	r := newTestRepo(t)
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

	ds := &dataset.Dataset{
		Name: "animals",
		Structure: &dataset.Structure{
			Format: "csv",
		},
	}
	ds.SetBodyFile(qfs.NewMemfileBytes("animals.csv",
		[]byte("Animal,Sound,Weight\ncat,meow,1.4\ndog,bark,3.7\n")))
	if err = InferValues(pro, ds); err != nil {
		t.Error(err)
	}

	if ds.Structure.Format != "csv" {
		t.Errorf("expected format CSV, got %s", ds.Structure.Format)
	}
	if ds.Structure.FormatConfig["headerRow"] != true {
		t.Errorf("expected format config to set headerRow set to true")
	}

	actual := datasetSchemaToJSON(ds)
	expect := `{"items":{"items":[{"title":"animal","type":"string"},{"title":"sound","type":"string"},{"title":"weight","type":"number"}],"type":"array"},"type":"array"}`

	if expect != actual {
		t.Errorf("mismatched schema, expected \"%s\", got \"%s\"", expect, actual)
	}
}

func TestInferValuesDontOverwriteSchema(t *testing.T) {
	r := newTestRepo(t)
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

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
	if err = InferValues(pro, ds); err != nil {
		t.Error(err)
	}

	if ds.Structure.Format != "csv" {
		t.Errorf("expected format CSV, got %s", ds.Structure.Format)
	}
	if ds.Structure.FormatConfig != nil {
		t.Errorf("expected format config to be nil")
	}

	actual := datasetSchemaToJSON(ds)
	expect := `{"items":{"items":[{"title":"animal","type":"number"},{"title":"noise","type":"number"},{"title":"height","type":"number"}],"type":"array"},"type":"array"}`

	if expect != actual {
		t.Errorf("mismatched schema, expected \"%s\", got \"%s\"", expect, actual)
	}
}

func TestMaybeAddDefaultViz(t *testing.T) {
	r := newTestRepo(t)
	_, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

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

func TestValidateDataset(t *testing.T) {
	if err := ValidateDataset(&dataset.Dataset{Name: "this name has spaces"}); err == nil {
		t.Errorf("expected invalid name to fail")
	}
}

func datasetSchemaToJSON(ds *dataset.Dataset) string {
	js, err := json.Marshal(ds.Structure.Schema)
	if err != nil {
		return err.Error()
	}
	return string(js)
}
