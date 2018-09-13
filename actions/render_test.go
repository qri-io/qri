package actions

import (
	"testing"
)

func TestRender(t *testing.T) {
	node := newTestNode(t)
	ref := addCitiesDataset(t, node)

	_, err := Render(node.Repo, ref, nil, 0, 0, true)
	if err != nil {
		t.Error(err.Error())
	}

}
