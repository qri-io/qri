package base

import (
	"testing"
)

func TestListDatasets(t *testing.T) {
	r := newTestRepo(t)
	addCitiesDataset(t, r)

	res, err := ListDatasets(r, 1, 0, false)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 1 {
		t.Error("expected one dataset response")
	}
}
