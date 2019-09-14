package actions

import (
	"context"
	"testing"
)

func TestDatasetHead(t *testing.T) {
	ctx := context.Background()
	node := newTestNode(t)
	ref := addCitiesDataset(t, node)

	if err := DatasetHead(ctx, node, &ref); err != nil {
		t.Error(err.Error())
	}
	if ref.Dataset == nil {
		t.Error("expected dataset to be populated")
	}
}
