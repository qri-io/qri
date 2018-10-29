package base

import (
	"testing"
)

func TestDatasetLog(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)
	log, err := DatasetLog(r, ref, 100, 0)
	if err != nil {
		t.Error(err)
	}
	if len(log) != 1 {
		t.Errorf("log length mismatch. expected: %d, got: %d", 1, len(log))
	}
}
