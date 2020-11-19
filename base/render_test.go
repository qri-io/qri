package base

import (
	"context"
	"testing"
)

func TestRender(t *testing.T) {
	t.Skip("TODO (b5) - need to fix qfs / repo connection for this to work")
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	ds, err := NewLocalDatasetLoader(r.Filesystem()).LoadDataset(ctx, ref, "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = Render(ctx, r, ds, nil)
	if err != nil {
		t.Error(err.Error())
	}

}
