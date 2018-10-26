package base

import (
	"testing"
)

func TestListDatasets(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	res, err := ListDatasets(r, 1, 0, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 1 {
		t.Error("expected one dataset response")
	}

	res, err = ListDatasets(r, 1, 0, false, true)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 0 {
		t.Error("expected no published datasets")
	}

	ref.Published = true
	if err := SetPublishStatus(r, &ref); err != nil {
		t.Fatal(err)
	}

	res, err = ListDatasets(r, 1, 0, false, true)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 1 {
		t.Error("expected one published dataset response")
	}
}
