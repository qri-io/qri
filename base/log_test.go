package base

import (
	"testing"

	"github.com/qri-io/qri/repo"
)

func TestDatasetLog(t *testing.T) {
	r := newTestRepo(t)
	addCitiesDataset(t, r)
	head := updateCitiesDataset(t, r)
	expectLen := 2

	dlog, err := DatasetLog(r, head, 100, 0, true)
	if err != nil {
		t.Error(err)
	}
	if len(dlog) != expectLen {
		t.Errorf("log length mismatch. expected: %d, got: %d", expectLen, len(dlog))
	}
	if dlog[0].Dataset.Meta.Title != head.Dataset.Meta.Title {
		t.Errorf("expected log with loadDataset == true to populate datasets")
	}

	dlog, err = DatasetLog(r, head, 100, 0, false)
	if err != nil {
		t.Error(err)
	}

	if len(dlog) != expectLen {
		t.Errorf("log length mismatch. expected: %d, got: %d", expectLen, len(dlog))
	}
	if dlog[0].Dataset.Meta.Title != "" {
		t.Errorf("expected log with loadDataset == false to not load a dataset. got: %v", dlog[0].Dataset)
	}
}

func TestLogDiff(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)
	head := updateCitiesDataset(t, r)

	ldr, err := LogDiff(r, []repo.DatasetRef{})
	if err == nil {
		t.Error("expected empty diff to error")
	}

	ldr, err = LogDiff(r, []repo.DatasetRef{
		repo.DatasetRef{Peername: "missing", Name: "reference"},
	})
	if err == nil {
		t.Error("expected diff of missing reference to error")
	}

	ldr, err = LogDiff(r, []repo.DatasetRef{ref})
	if err != nil {
		t.Error(err)
	}

	expectAdd := []repo.DatasetRef{head}
	if !RefSetEqual(ldr.Add, expectAdd) {
		t.Errorf("add mismatch. expected: %v\ngot: %v", expectAdd, ldr.Add)
	}
}

func TestRefDiff(t *testing.T) {
	a := []repo.DatasetRef{
		repo.MustParseDatasetRef("a/b@c/ipfs/d"),
		repo.MustParseDatasetRef("a/b@c/ipfs/e"),
		repo.MustParseDatasetRef("a/b@c/ipfs/f"),
	}
	b := []repo.DatasetRef{
		repo.MustParseDatasetRef("a/b@c/ipfs/e"),
		repo.MustParseDatasetRef("a/b@c/ipfs/f"),
		repo.MustParseDatasetRef("a/b@c/ipfs/g"),
		repo.MustParseDatasetRef("a/b@c/ipfs/h"),
	}
	expectAdd := []repo.DatasetRef{
		repo.MustParseDatasetRef("a/b@c/ipfs/g"),
		repo.MustParseDatasetRef("a/b@c/ipfs/h"),
	}
	expectRm := []repo.DatasetRef{
		repo.MustParseDatasetRef("a/b@c/ipfs/d"),
	}

	gotAdd, gotRm := refDiff(a, b)

	if !RefSetEqual(expectAdd, gotAdd) {
		t.Errorf("add mismatch. expected: %v\ngot: %v", expectAdd, gotAdd)
	}
	if !RefSetEqual(expectRm, gotRm) {
		t.Errorf("remove mismatch. expected: %v\ngot: %v", expectRm, gotRm)
	}
}

func RefSetEqual(a, b []repo.DatasetRef) bool {
	if len(a) != len(b) {
		return false
	}
	for i, ar := range a {
		if !b[i].Equal(ar) {
			return false
		}
	}
	return true
}
