package spec

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
)

// PutDatasetFunc adds a dataset to a system that stores datasets
// PutDatasetFunc is required to run the LoaderSpec test. When called the
// Loader should retain the dataset for later loading by the spec test
type PutDatasetFunc func(ds *dataset.Dataset) (path string, err error)

// AssertLoaderSpec confirms the expected behaviour of a dsref.Loader
// Interface implementation. In addition to this test passing, implementations
// MUST be nil-callable. Please add a nil-callable test for each implementation
//
// TODO(b5) - loader spec is intentionally a little vague at the moment. I'm not
// sure the interface belongs in this package, and this test isn't working
// network sources. This test serves to confirm basic requirements of a local
// loader function for the moment
func AssertLoaderSpec(t *testing.T, r dsref.Loader, putFunc PutDatasetFunc) {
	t.Helper()

	var (
		ctx      = context.Background()
		username = "example_user"
		name     = "this_is_an_example"
		path     = ""
	)

	ds, err := GenerateExampleDataset(ctx)
	if err != nil {
		t.Fatal(err)
	}

	path, err = putFunc(ds)
	if err != nil {
		t.Fatalf("putting dataset: %s", err)
	}

	_, err = r.LoadDataset(ctx, "")
	if err == nil {
		t.Errorf("expected loading without a reference Path value to fail, got nil")
	}

	ref := dsref.Ref{
		Username: username,
		Name:     name,
		Path:     path,
	}
	got, err := r.LoadDataset(ctx, ref.String())
	if err != nil {
		t.Fatal(err)
	}

	if got.BodyFile() == nil {
		t.Errorf("expected body file to be open & ready to read")
	}

	if username != got.Peername {
		t.Errorf("load Dataset didn't set dataset.Peername field to given reference. want: %q got: %q", username, got.Peername)
	}
	if name != got.Name {
		t.Errorf("load Dataset didn't set dataset.Name field to given reference. want: %q got: %q", name, got.Name)
	}

	ds.Peername = username
	ds.Name = name
	ds.Path = path
	if diff := cmp.Diff(ds, got, cmpopts.IgnoreUnexported(dataset.Dataset{}, dataset.Meta{})); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

// GenerateExampleDataset creates an example dataset document
func GenerateExampleDataset(ctx context.Context) (*dataset.Dataset, error) {
	ds := &dataset.Dataset{
		Commit: &dataset.Commit{
			Title: "initial commit",
		},
		Meta: &dataset.Meta{
			Title:       "dataset loader spec test",
			Description: "a dataset to check that",
		},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "array",
					"items": []interface{}{
						map[string]interface{}{"title": "a", "type": "string"},
						map[string]interface{}{"title": "b", "type": "number"},
						map[string]interface{}{"title": "c", "type": "boolean"},
					},
				},
			},
		},
	}

	ds.SetBodyFile(qfs.NewMemfileBytes("body.json", []byte(`[
		["a",1,false],
		["b",2,true],
		["c",3,true]
	]`)))

	return ds, nil
}
