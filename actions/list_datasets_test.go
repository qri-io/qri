package actions

import (
	"testing"

	"github.com/qri-io/qri/repo"
)

func TestListDatasets(t *testing.T) {
	node := newTestNode(t)
	addCitiesDataset(t, node)

	res, err := ListDatasets(node, &repo.DatasetRef{Peername: "me"}, 1, 0, false)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 1 {
		t.Error("expected one dataset response")
	}
}
