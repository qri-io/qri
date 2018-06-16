package lib

import (
	"testing"

	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestSelectionRequestsSelectedRefs(t *testing.T) {
	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}

	sr := NewSelectionRequests(mr, nil)
	var done bool
	a := []repo.DatasetRef{{Peername: "a"}}
	if err := sr.SetSelectedRefs(&a, &done); err != nil {
		t.Errorf("setting selected refs: %s", err.Error())
	}

	b := []repo.DatasetRef{}
	if err := sr.SelectedRefs(&done, &b); err != nil {
		t.Errorf("selected refs: %s", err.Error())
	}

	if len(a) != len(b) {
		t.Errorf("repsonse len mismatch. expected: %d. got: %d", len(a), len(b))
	}
	for i, ar := range a {
		if err := repo.CompareDatasetRef(ar, b[i]); err != nil {
			t.Errorf("case selection %d error: %s", i, err)
		}
	}
}
