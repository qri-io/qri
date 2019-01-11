package base

import (
	"testing"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
)

func TestPrepareDatasetSave(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	_, _, _, _, err := PrepareDatasetSave(r, ref.Peername, ref.Name)
	if err != nil {
		t.Error(err.Error())
	}

	_, _, _, _, err = PrepareDatasetSave(r, "me", "non-existent")
	if err != nil {
		t.Error(err.Error())
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
