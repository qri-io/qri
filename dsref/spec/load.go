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
// Loader should retain the dataset for later loading by the spec test, and
// return a full reference to the saved version
type PutDatasetFunc func(ds *dataset.Dataset) (ref *dsref.Ref, err error)

// AssertLoaderSpec confirms the expected behaviour of a dsref.Loader
// Interface implementation. In addition to this test passing, implementations
// MUST be nil-callable. Please add a nil-callable test for each implementation
//
// TODO(b5) - this test isn't working network sources. At the moment it confirms
// only basic requirements of a local loader function
func AssertLoaderSpec(t *testing.T, r dsref.Loader, putFunc PutDatasetFunc) {
	ctx := context.Background()

	ds, err := GenerateExampleDataset(ctx)
	if err != nil {
		t.Fatal(err)
	}

	loadableRef, err := putFunc(ds)
	if err != nil {
		t.Fatalf("putting dataset: %s", err)
	}

	// update our expected dataset with values from the added ref
	ds.Peername = loadableRef.Username
	ds.Name = loadableRef.Name
	ds.Path = loadableRef.Path
	// TODO(b5): loading should require initID be set, but this breaks *many* tests
	// ds.ID = loadableRef.InitID

	t.Run("edge_cases", func(t *testing.T) {
		if _, err := r.LoadDataset(ctx, ""); err == nil {
			t.Errorf("expected loading without a reference Path value to fail, got nil")
		}
	})

	t.Run("full_reference_provided", func(t *testing.T) {

		got, err := r.LoadDataset(ctx, loadableRef.String())
		if err != nil {
			t.Fatal(err)
		}

		if got.BodyFile() == nil {
			t.Errorf("expected body file to be open & ready to read")
		}

		if loadableRef.Username != got.Peername {
			t.Errorf("load Dataset didn't set dataset.Peername field to given reference. want: %q got: %q", loadableRef.Username, got.Peername)
		}
		if loadableRef.Name != got.Name {
			t.Errorf("load Dataset didn't set dataset.Name field to given reference. want: %q got: %q", loadableRef.Name, got.Name)
		}

		if diff := cmp.Diff(ds, got, cmpopts.IgnoreUnexported(dataset.Dataset{}, dataset.Meta{})); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("alias_provided", func(t *testing.T) {
		got, err := r.LoadDataset(ctx, loadableRef.Alias())
		if err != nil {
			t.Fatal(err)
		}

		if got.BodyFile() == nil {
			t.Errorf("expected body file to be open & ready to read")
		}

		if loadableRef.Username != got.Peername {
			t.Errorf("load Dataset didn't set dataset.Peername field to given reference. want: %q got: %q", loadableRef.Username, got.Peername)
		}
		if loadableRef.Name != got.Name {
			t.Errorf("load Dataset didn't set dataset.Name field to given reference. want: %q got: %q", loadableRef.Name, got.Name)
		}

		if diff := cmp.Diff(ds, got, cmpopts.IgnoreUnexported(dataset.Dataset{}, dataset.Meta{})); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
	})
}

// GenerateExampleDataset creates an example dataset document
func GenerateExampleDataset(ctx context.Context) (*dataset.Dataset, error) {
	ds := &dataset.Dataset{
		Name: "example_loader_spec_test",
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
