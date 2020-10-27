package base

import (
	"context"
	"testing"

	"github.com/qri-io/qri/dsref"
)

func TestSelect(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	if _, err := Select(ctx, r.Filesystem(), dsref.Ref{Username: "bad", Name: "ref"}, "commit"); err == nil {
		t.Error("expected select of bad ref to fail")
	}
	if _, err := Select(ctx, r.Filesystem(), ref, ""); err != nil {
		t.Error(err.Error())
	}
	if _, err := Select(ctx, r.Filesystem(), ref, "commit"); err != nil {
		t.Error(err.Error())
	}
	if _, err := Select(ctx, r.Filesystem(), ref, "meta.title"); err != nil {
		t.Error(err.Error())
	}
	if _, err := Select(ctx, r.Filesystem(), ref, "structure.schema.items.0"); err != nil {
		t.Error(err.Error())
	}
}

func TestApplyPath(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	ds, err := ReadDataset(ctx, r, ref.Path)
	if err != nil {
		t.Error(err)
	}

	body, err := ApplyPath(ds, "meta.title")
	if err != nil {
		t.Error(err)
	}
	if body == nil {
		t.Error("expected body to not be nil")
	}
}
