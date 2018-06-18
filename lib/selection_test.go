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

func TestDefaultSelectedRefs(t *testing.T) {
	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Fatalf("allocating test repo: %s", err)
	}

	refs := []repo.DatasetRef{}
	if err := DefaultSelectedRefs(mr, &refs); err != nil {
		t.Error(err.Error())
	}
	if len(refs) != 0 {
		t.Error("expected 0 references")
	}

	sr := NewSelectionRequests(mr, nil)
	var done bool
	a := []repo.DatasetRef{{Peername: "a"}, {Peername: "b"}}
	if err := sr.SetSelectedRefs(&a, &done); err != nil {
		t.Errorf("setting selected refs: %s", err.Error())
	}

	if err := DefaultSelectedRefs(mr, &refs); err != nil {
		t.Error(err.Error())
	}
	if len(refs) != 2 {
		t.Error("expected 2 references")
	}

	refs = []repo.DatasetRef{}
	b := []repo.DatasetRef{}
	if err := sr.SetSelectedRefs(&b, &done); err != nil {
		t.Errorf("setting selected refs: %s", err.Error())
	}
	if err := DefaultSelectedRefs(mr, &refs); err != nil {
		t.Error(err.Error())
	}
	if len(refs) != 0 {
		t.Error("expected 0 references")
	}

}

func TestDefaultSelectedRef(t *testing.T) {
	mr, err := testrepo.NewTestRepo(nil)
	if err != nil {
		t.Fatalf("allocating test repo: %s", err)
	}

	ref := &repo.DatasetRef{}
	if err := DefaultSelectedRef(mr, ref); err != nil {
		t.Error(err.Error())
	}
	if !ref.IsEmpty() {
		t.Error("expected ref to be empty")
	}

	sr := NewSelectionRequests(mr, nil)
	var done bool
	a := []repo.DatasetRef{{Peername: "a"}, {Peername: "b"}}
	if err := sr.SetSelectedRefs(&a, &done); err != nil {
		t.Errorf("setting selected refs: %s", err.Error())
	}

	if err := DefaultSelectedRef(mr, ref); err != nil {
		t.Error(err.Error())
	}
	if ref.IsEmpty() {
		t.Error("expected ref not to be empty")
	}

	ref = &repo.DatasetRef{}
	b := []repo.DatasetRef{}
	if err := sr.SetSelectedRefs(&b, &done); err != nil {
		t.Errorf("setting selected refs: %s", err.Error())
	}
	if err := DefaultSelectedRef(mr, ref); err != nil {
		t.Error(err.Error())
	}
	if !ref.IsEmpty() {
		t.Error("expected ref to be empty")
	}

}
