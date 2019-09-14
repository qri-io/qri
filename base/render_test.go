package base

import (
	"context"
	"testing"
)

func TestRender(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	_, err := Render(ctx, r, ref, nil)
	if err != nil {
		t.Error(err.Error())
	}

}
