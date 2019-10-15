package base

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
)

func TestDatasetLogFromHistory(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	addCitiesDataset(t, r)
	head := updateCitiesDataset(t, r)
	expectLen := 2

	dlog, err := DatasetLogFromHistory(ctx, r, head, 0, 100, true)
	if err != nil {
		t.Error(err)
	}
	if len(dlog) != expectLen {
		t.Fatalf("log length mismatch. expected: %d, got: %d", expectLen, len(dlog))
	}
	if dlog[0].Dataset.Meta.Title != head.Dataset.Meta.Title {
		t.Errorf("expected log with loadDataset == true to populate datasets")
	}

	dlog, err = DatasetLogFromHistory(ctx, r, head, 0, 100, false)
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
	ref := updateCitiesDataset(t, r)

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

	cities := repo.ConvertToDsref(ref)

	// confirm no history exists:
	if _, err = book.Versions(cities, 0, 100); err == nil {
		t.Errorf("expected versions for nonexistent history to fail")
	}

	// create some history
	if err := constructDatasetLogFromHistory(ctx, r, cities); err != nil {
		t.Errorf("building dataset history: %s", err)
	}

	expect := []logbook.DatasetInfo{
		{
			CommitTitle: "initial commit",
		},
		{
			CommitTitle: "initial commit",
		},
	}

	// confirm history exists:
	got, err := book.Versions(cities, 0, 100)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(expect, got, cmpopts.IgnoreFields(logbook.DatasetInfo{}, "Timestamp", "Ref")); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
