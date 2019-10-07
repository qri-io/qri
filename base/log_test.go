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
