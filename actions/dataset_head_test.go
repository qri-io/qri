package actions

import "testing"

func TestDatasetHead(t *testing.T) {
	node := newTestNode(t)
	ref := addCitiesDataset(t, node)

	if err := DatasetHead(node, &ref); err != nil {
		t.Error(err.Error())
	}
	if ref.Dataset == nil {
		t.Error("expected dataset to be populated")
	}
}
