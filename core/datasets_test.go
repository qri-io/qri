package core

import (
	"testing"

	"github.com/qri-io/dataset"
)

func TestDatasetRequestsInit(t *testing.T) {
	cases := []struct {
		p   *InitDatasetParams
		res *dataset.Dataset
		err string
	}{
		{&InitDatasetParams{}, nil, "data file is required"},
		{&InitDatasetParams{Data: badDataFile}, nil, "error determining dataset schema: line 3, column 0: wrong number of fields in line"},
		{&InitDatasetParams{Data: jobsByAutomationFile}, nil, ""},
	}

	mr, ms, err := NewTestRepo()
	if err != nil {
		t.Errorf("error allocating test repo: %s", err.Error())
		return
	}

	req := NewDatasetRequests(ms, mr)
	for i, c := range cases {
		got := &dataset.Dataset{}
		err := req.InitDataset(c.p, got)

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch: expected: %s, got: %s", i, c.err, err)
			continue
		}
	}
}
