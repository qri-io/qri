package actions

import (
	"context"
	"testing"
)

func TestRecall(t *testing.T) {
	ctx := context.Background()
	node := newTestNode(t)
	ref := addNowTransformDataset(t, node)

	_, err := Recall(ctx, node, "", ref)
	if err != nil {
		t.Error(err)
	}

	_, err = Recall(ctx, node, "tf", ref)
	if err != nil {
		t.Error(err)
	}
}
