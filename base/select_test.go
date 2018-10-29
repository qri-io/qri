package base

import (
	"testing"

	"github.com/qri-io/qri/repo"
)

func TestSelect(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	if _, err := Select(r, repo.DatasetRef{Peername: "bad", Name: "ref"}, "commit"); err == nil {
		t.Error("expected select of bad ref to fail")
	}
	if _, err := Select(r, ref, ""); err != nil {
		t.Error(err.Error())
	}
	if _, err := Select(r, ref, "commit"); err != nil {
		t.Error(err.Error())
	}
	if _, err := Select(r, ref, "meta.title"); err != nil {
		t.Error(err.Error())
	}
	if _, err := Select(r, ref, "structure.schema.items.0"); err != nil {
		t.Error(err.Error())
	}
}

func TestApplyPath(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	if err := ReadDataset(r, &ref); err != nil {
		t.Error(err)
	}

	body, err := ApplyPath(ref.Dataset, "meta.title")
	if err != nil {
		t.Error(err)
	}
	if body == nil {
		t.Error("expected body to not be nil")
	}
}
