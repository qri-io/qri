package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/repo"
)

// RepoMakerFunc produces a new instance of a repository when called
type RepoMakerFunc func(t *testing.T) repo.Repo

// repoTestFunc is a function for testing a repo
type repoTestFunc func(t *testing.T, rm RepoMakerFunc)

func testdataPath(path string) string {
	return filepath.Join(os.Getenv("GOPATH"), "/src/github.com/qri-io/qri/repo/test/testdata", path)
}

// RunRepoTests tests that this repo conforms to
// expected behaviors
func RunRepoTests(t *testing.T, rmf RepoMakerFunc) {
	tests := []repoTestFunc{
		testProfile,
		testCreateDataset,
		testReadDataset,
		testRenameDataset,
		testDatasetPinning,
		testDeleteDataset,
		testEventsLog,
		testRefstore,
	}

	for _, test := range tests {
		test(t, rmf)
	}
}

func testProfile(t *testing.T, rmf RepoMakerFunc) {
	r := rmf(t)
	p, err := r.Profile()
	if err != nil {
		t.Errorf("Unexpected Profile error: %s", err.Error())
		return
	}

	err = r.SaveProfile(p)
	if err != nil {
		t.Errorf("Unexpected SaveProfile error: %s", err.Error())
		return
	}

}

func testCreateDataset(t *testing.T, rmf RepoMakerFunc) {
	createDataset(t, rmf)
}

func createDataset(t *testing.T, rmf RepoMakerFunc) (repo.Repo, repo.DatasetRef) {
	r := rmf(t)
	r.SaveProfile(testPeerProfile)
	r.SetPrivateKey(privKey)

	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
	if err != nil {
		t.Error(err.Error())
		return r, repo.DatasetRef{}
	}

	ref, err := r.CreateDataset(tc.Name, tc.Input, tc.DataFile(), true)
	if err != nil {
		t.Error(err.Error())
	}

	return r, ref
}

func testReadDataset(t *testing.T, rmf RepoMakerFunc) {
	r, ref := createDataset(t, rmf)

	if err := r.ReadDataset(&ref); err != nil {
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

	b := repo.DatasetRef{
		Name:     "cities2",
		Path:     ref.Path,
		Peername: ref.Peername,
		PeerID:   ref.PeerID,
	}

	if err := r.RenameDataset(ref, b); err != nil {
		t.Error(err.Error())
		return
	}

	if err := r.ReadDataset(&b); err != nil {
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

	if err := r.PinDataset(ref); err != nil {
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

	ref2, err := r.CreateDataset(tc.Name, tc.Input, tc.DataFile(), false)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if err := r.PinDataset(ref2); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}

	if err := r.UnpinDataset(ref); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}

	if err := r.UnpinDataset(ref2); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}
}

func testDeleteDataset(t *testing.T, rmf RepoMakerFunc) {
	r, ref := createDataset(t, rmf)

	if err := r.DeleteDataset(ref); err != nil {
		t.Error(err.Error())
		return
	}
}

func testEventsLog(t *testing.T, rmf RepoMakerFunc) {
	r, ref := createDataset(t, rmf)
	pinner := true

	b := repo.DatasetRef{
		Name:     "cities2",
		Path:     ref.Path,
		Peername: ref.Peername,
		PeerID:   ref.PeerID,
	}

	if err := r.RenameDataset(ref, b); err != nil {
		t.Error(err.Error())
		return
	}

	if err := r.PinDataset(b); err != nil {
		if err == repo.ErrNotPinner {
			pinner = false
		} else {
			t.Error(err.Error())
			return
		}
	}

	if err := r.UnpinDataset(b); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}

	if err := r.DeleteDataset(b); err != nil {
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
