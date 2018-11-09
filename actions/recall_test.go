package actions

import "testing"

func TestRecall(t *testing.T) {
	node := newTestNode(t)
	ref := addNowTransformDataset(t, node)

	_, err := Recall(node, "", ref)
	if err != nil {
		t.Error(err)
	}

	_, err = Recall(node, "tf", ref)
	if err != nil {
		t.Error(err)
	}
}
