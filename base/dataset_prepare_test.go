package base

import (
	"testing"

	"github.com/qri-io/dataset/dstest"
)

func TestPrepareDatasetNew(t *testing.T) {
	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
	if err != nil {
		t.Fatal(err.Error())
	}

	dsp := tc.Input.Encode()
	dsp.BodyBytes = tc.Body

	_, _, _, err = PrepareDatasetNew(dsp)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestPrepareDatasetSave(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
	if err != nil {
		t.Fatal(err.Error())
	}

	dsp := tc.Input.Encode()
	dsp.Meta.Title = "updated"
	dsp.Name = ref.Name
	dsp.Peername = ref.Peername

	_, _, _, err = PrepareDatasetSave(r, dsp)
	if err != nil {
		t.Error(err.Error())
	}
}
