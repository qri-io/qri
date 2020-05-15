package spec

import (
	"testing"

	"github.com/qri-io/qri/dsref"
)

func TestResolverConsistency(t *testing.T) {
	a, b := dsref.NewMemResolver("a"), dsref.NewMemResolver("b")
	ref := dsref.Ref{
		InitID:   "a_thing_that_does_not_change",
		Username: "a",
		Name:     "example_dataset",
		Path:     "/ipfs/QmExample",
	}

	toResolve := dsref.Ref{
		Username: "a",
		Name:     "example_dataset",
	}

	if err := ConsistentResolvers(t, toResolve, a, b); err != nil {
		t.Error(err)
	}

	a.Put(dsref.VersionInfo{
		InitID:   ref.InitID,
		Username: ref.Username,
		Name:     ref.Name,
		Path:     ref.Path,
	})

	if err := InconsistentResolvers(t, toResolve, a, b); err != nil {
		t.Error(err)
	}

	b.Put(dsref.VersionInfo{
		InitID:   ref.InitID,
		Username: ref.Username,
		Name:     ref.Name,
		Path:     ref.Path,
	})

	if err := ConsistentResolvers(t, toResolve, a, b); err != nil {
		t.Error(err)
	}

	if err := InconsistentResolvers(t, toResolve, a, b); err == nil {
		t.Error("expected error, got nil")
	}

	b.Put(dsref.VersionInfo{
		InitID:   ref.InitID,
		Username: ref.Username,
		Name:     ref.Name,
		Path:     "/fsi/local/checkout/path",
	})

	if err := ConsistentResolvers(t, toResolve, a, b); err != nil {
		t.Error(err)
	}

	c := dsref.NewMemResolver("c")
	if err := InconsistentResolvers(t, toResolve, b, c); err != nil {
		t.Error(err)
	}

	c.Put(dsref.VersionInfo{
		InitID:   "incorrect_id",
		Username: "a",
		Name:     "example_dataset",
		Path:     "/ipfs/QmBadExample",
	})

	if err := InconsistentResolvers(t, toResolve, b, c); err != nil {
		t.Error(err)
	}
	if err := InconsistentResolvers(t, toResolve, a, c); err != nil {
		t.Error(err)
	}
}
