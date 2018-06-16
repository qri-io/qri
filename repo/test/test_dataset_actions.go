package test

import (
	"testing"

	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/actions"
)

// DatasetActions runs actions.Dataset tests against a given repo
// TODO - actions should only use public repo methods.
// remove this test in favor of a lower-level test suite
func DatasetActions(t *testing.T, rmf RepoMakerFunc) {
	for _, test := range []repoTestFunc{
		testCreateDataset,
		testReadDataset,
		testRenameDataset,
		testDatasetPinning,
		testDeleteDataset,
		testEventsLog,
	} {
		test(t, rmf)
	}
}

func testCreateDataset(t *testing.T, rmf RepoMakerFunc) {
	createDataset(t, rmf)
}

func createDataset(t *testing.T, rmf RepoMakerFunc) (repo.Repo, repo.DatasetRef) {
	r := rmf(t)
	r.SetProfile(testPeerProfile)
	act := actions.Dataset{r}

	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
	if err != nil {
		t.Error(err.Error())
		return r, repo.DatasetRef{}
	}

	ref, err := act.CreateDataset(tc.Name, tc.Input, tc.BodyFile(), nil, true)
	if err != nil {
		t.Error(err.Error())
	}

	return r, ref
}

func testReadDataset(t *testing.T, rmf RepoMakerFunc) {
	r, ref := createDataset(t, rmf)
	act := actions.Dataset{r}

	if err := act.ReadDataset(&ref); err != nil {
		t.Error(err.Error())
		return
	}

	if ref.Dataset == nil {
		t.Error("expected dataset to not equal nil")
		return
	}
}

func testRenameDataset(t *testing.T, rmf RepoMakerFunc) {
	r, ref := createDataset(t, rmf)
	act := actions.Dataset{r}

	b := repo.DatasetRef{
		Name:      "cities2",
		Path:      ref.Path,
		Peername:  ref.Peername,
		ProfileID: ref.ProfileID,
	}

	if err := act.RenameDataset(ref, b); err != nil {
		t.Error(err.Error())
		return
	}

	if err := act.ReadDataset(&b); err != nil {
		t.Error(err.Error())
		return
	}

	if b.Dataset == nil {
		t.Error("expected dataset to not equal nil")
		return
	}
}

func testDatasetPinning(t *testing.T, rmf RepoMakerFunc) {
	r, ref := createDataset(t, rmf)
	act := actions.Dataset{r}

	if err := act.PinDataset(ref); err != nil {
		if err == repo.ErrNotPinner {
			t.Log("repo store doesn't support pinning")
		} else {
			t.Error(err.Error())
			return
		}
	}

	tc, err := dstest.NewTestCaseFromDir(testdataPath("counter"))
	if err != nil {
		t.Error(err.Error())
		return
	}

	ref2, err := act.CreateDataset(tc.Name, tc.Input, tc.BodyFile(), nil, false)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if err := act.PinDataset(ref2); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}

	if err := act.UnpinDataset(ref); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}

	if err := act.UnpinDataset(ref2); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}
}

func testDeleteDataset(t *testing.T, rmf RepoMakerFunc) {
	r, ref := createDataset(t, rmf)
	act := actions.Dataset{r}

	if err := act.DeleteDataset(ref); err != nil {
		t.Error(err.Error())
		return
	}
}

func testEventsLog(t *testing.T, rmf RepoMakerFunc) {
	r, ref := createDataset(t, rmf)
	act := actions.Dataset{r}
	pinner := true

	b := repo.DatasetRef{
		Name:      "cities2",
		Path:      ref.Path,
		Peername:  ref.Peername,
		ProfileID: ref.ProfileID,
	}

	if err := act.RenameDataset(ref, b); err != nil {
		t.Error(err.Error())
		return
	}

	if err := act.PinDataset(b); err != nil {
		if err == repo.ErrNotPinner {
			pinner = false
		} else {
			t.Error(err.Error())
			return
		}
	}

	if err := act.UnpinDataset(b); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}

	if err := act.DeleteDataset(b); err != nil {
		t.Error(err.Error())
		return
	}

	events, err := r.Events(10, 0)
	if err != nil {
		t.Error(err.Error())
		return
	}

	ets := []repo.EventType{repo.ETDsDeleted, repo.ETDsUnpinned, repo.ETDsPinned, repo.ETDsRenamed, repo.ETDsPinned, repo.ETDsCreated}

	if !pinner {
		ets = []repo.EventType{repo.ETDsDeleted, repo.ETDsRenamed, repo.ETDsCreated}
	}

	if len(events) != len(ets) {
		t.Errorf("event log length mismatch. expected: %d, got: %d", len(ets), len(events))
		return
	}

	for i, et := range ets {
		if events[i].Type != et {
			t.Errorf("case %d eventType mismatch. expected: %s, got: %s", i, et, events[i].Type)
		}
	}
}
