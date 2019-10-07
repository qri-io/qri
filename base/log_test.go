package base

import (
	"context"
	"testing"

	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
)

func TestDatasetLog(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	addCitiesDataset(t, r)
	head := updateCitiesDataset(t, r)
	expectLen := 2

	dlog, err := DatasetLog(ctx, r, head, 100, 0, true)
	if err != nil {
		t.Error(err)
	}
	if len(dlog) != expectLen {
		t.Errorf("log length mismatch. expected: %d, got: %d", expectLen, len(dlog))
	}
	if dlog[0].Dataset.Meta.Title != head.Dataset.Meta.Title {
		t.Errorf("expected log with loadDataset == true to populate datasets")
	}

	dlog, err = DatasetLog(ctx, r, head, 100, 0, false)
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
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)
	head := updateCitiesDataset(t, r)

	ldr, err := LogDiff(ctx, r, []repo.DatasetRef{})
	if err == nil {
		t.Error("expected empty diff to error")
	}

	ldr, err = LogDiff(ctx, r, []repo.DatasetRef{
		repo.DatasetRef{Peername: "missing", Name: "reference"},
	})
	if err == nil {
		t.Error("expected diff of missing reference to error")
	}

	ldr, err = LogDiff(ctx, r, []repo.DatasetRef{ref})
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

func TestConstructDatasetLogFromHistory(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t).(*repo.MemRepo)

	// remove the logbook
	r.RemoveLogbook()

	// create some history
	addCitiesDataset(t, r)
	updateCitiesDataset(t, r)

	// add the logbook back
	p, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}
	book, err := logbook.NewBook(p.PrivKey, p.Peername, r.Filesystem(), "/map/logbook")
	if err != nil {
		t.Fatal(err)
	}
	r.SetLogbook(book)

	ref := dsref.Ref{ Username: p.Peername, Name: "cities" }

	// confirm no history exists:
	if _, err = book.Versions(ref, 0, 100); err == nil {
		t.Errorf("expected versions for nonexistent history to fail")
	}
	
	// create some history
	if err := ConstructDatasetLogFromHistory(ctx, r, ref); err != nil {
		t.Errorf("building dataset history: %s", err)
	}

	// confirm history exists:
	if _, err = book.Versions(ref, 0, 100); err != nil {
		t.Error(err)
	}
}
