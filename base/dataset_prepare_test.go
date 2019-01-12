package base

import (
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
)

func TestPrepareDatasetSave(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	prev, mutable, body, prevPath, err := PrepareDatasetSave(r, ref.Peername, ref.Name)
	if err != nil {
		t.Errorf("case cities dataset error: %s ", err.Error())
	}
	if prev.IsEmpty() {
		t.Errorf("case cites dataset: previous should not be empty")
	}
	if mutable.IsEmpty() {
		t.Errorf("case cities dataset: mutable should not be empty")
	}
	if mutable.Transform != nil {
		t.Errorf("case cities dataset: mutable.Transform should be nil")
	}
	if mutable.Commit != nil {
		t.Errorf("case cities dataset: mutable.Commit should be nil")
	}
	if body == nil {
		t.Errorf("case cities dataset: previous body should not be nil")
	}
	if prevPath == "" {
		t.Errorf("case cities dataset: previous path should not be empty")
	}

	prev, mutable, body, prevPath, err = PrepareDatasetSave(r, "me", "non-existent")
	if err != nil {
		t.Errorf("case non-existant previous dataset error: %s ", err.Error())
	}
	if !prev.IsEmpty() {
		t.Errorf("case non-existant previous dataset: previous should be empty, got non-empty dataset")
	}
	if !mutable.IsEmpty() {
		t.Errorf("case non-existant previous dataset: mutable should be empty, got non-empty dataset")
	}
	if body != nil {
		t.Errorf("case non-existant previous dataset: previous body should be nil, got non-nil body")
	}
	if prevPath != "" {
		t.Errorf("case non-existant previous dataset: previous path should be empty, got non-empty path")
	}
}

func TestInferValues(t *testing.T) {
	r := newTestRepo(t)
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

	name := ""
	body := cafs.NewMemfileBytes("gabba gabba hey.csv", []byte("a,b,c,c,s,v"))
	ds := &dataset.Dataset{}
	if _, err = InferValues(pro, &name, ds, body); err != nil {
		t.Error(err)
	}
	// TODO - lol fix varname, so broken
	expectName := "gabba_gabba_heycsv"
	if expectName != name {
		t.Errorf("inferred name mismatch. expected: '%s', got: '%s'", expectName, name)
	}
}

func TestValidateDataset(t *testing.T) {
	if err := ValidateDataset("this name has spaces", nil); err == nil {
		t.Errorf("expected invalid name to fail")
	}
}
