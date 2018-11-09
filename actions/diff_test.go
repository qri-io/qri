package actions

import (
	"testing"
)

func TestDiffDatasets(t *testing.T) {
	node := newTestNode(t)
	cities := addCitiesDataset(t, node)
	fc := addFlourinatedCompoundsDataset(t, node)

	diffs, err := DiffDatasets(node, cities, fc, true, nil)
	if err != nil {
		t.Error(err.Error())
	}
	if len(diffs) == 0 {
		t.Error("expected some diffs")
	}
}
