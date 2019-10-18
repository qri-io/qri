package actions

import (
	"context"
	"testing"
)

func TestDiffDatasets(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	cities := addCitiesDataset(t, r)
	fc := addFlourinatedCompoundsDataset(t, r)

	diffs, err := DiffDatasets(ctx, r, cities, fc, true, nil)
	if err != nil {
		t.Error(err.Error())
	}
	if len(diffs) == 0 {
		t.Error("expected some diffs")
	}
}
