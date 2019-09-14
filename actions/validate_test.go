package actions

import (
	"context"
	"testing"
)

func TestValidate(t *testing.T) {
	ctx := context.Background()
	node := newTestNode(t)
	cities := addCitiesDataset(t, node)

	errs, err := Validate(ctx, node, cities, nil, nil)
	if err != nil {
		t.Error(err.Error())
	}

	if len(errs) != 0 {
		t.Errorf("expected 0 errors. got: %d", len(errs))
	}
}
