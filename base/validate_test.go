package base

import (
	"context"
	"testing"
)

func TestValidate(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	cities := addCitiesDataset(t, r)

	errs, err := Validate(ctx, r, cities, nil, nil)
	if err != nil {
		t.Error(err.Error())
	}

	if len(errs) != 0 {
		t.Errorf("expected 0 errors. got: %d", len(errs))
	}
}
