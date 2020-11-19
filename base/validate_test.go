package base

import (
	"context"
	"testing"

	"github.com/qri-io/qri/base/dsfs"
)

func TestValidate(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	cities := addCitiesDataset(t, r)
	fs := r.Filesystem()

	ds, err := dsfs.LoadDataset(ctx, fs, cities.Path)
	if err != nil {
		t.Fatal(err)
	}

	if err = OpenDataset(ctx, fs, ds); err != nil {
		t.Fatal(err)
	}
	body := ds.BodyFile()

	errs, err := Validate(ctx, r, body, ds.Structure)
	if err != nil {
		t.Error(err.Error())
	}

	if len(errs) != 0 {
		t.Errorf("expected 0 errors. got: %d", len(errs))
	}
}
