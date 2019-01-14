package actions

import (
	"testing"

	"github.com/qri-io/qri/repo"
)

func TestListDatasets(t *testing.T) {
	node := newTestNode(t)
	addCitiesDataset(t, node)

	res, err := ListDatasets(node, &repo.DatasetRef{Peername: "me"}, 1, 0, false, false)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 1 {
		t.Error("expected one dataset response")
	}
}

func TestListDatasetsNotFound(t *testing.T) {
	node := newTestNode(t)
	addCitiesDataset(t, node)

	_, err := ListDatasets(node, &repo.DatasetRef{Peername: "not_found"}, 1, 0, false, false)
	if err == nil {
		t.Error("expected to get error")
	}
	expect := "profile not found: \"not_found\""
	if expect != err.Error() {
		t.Errorf("expected error \"%s\", got \"%s\"", expect, err.Error())
	}
}
