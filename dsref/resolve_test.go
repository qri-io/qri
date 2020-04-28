package dsref

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMemResolver(t *testing.T) {
	ctx := context.Background()
	m := NewMemResolver("test_peer")
	m.Put(VersionInfo{
		InitID:   "myInitID",
		Username: "test_peer",
		Name:     "my_ds",
		Path:     "/ipfs/QmeXaMpLe",
	})

	if _, err := (*MemResolver)(nil).ResolveRef(ctx, nil); err != ErrNotFound {
		t.Errorf("book ResolveRef must be nil-callable. expected: %q, got %v", ErrNotFound, err)
	}

	if _, err := m.ResolveRef(ctx, &Ref{Username: "username", Name: "does_not_exist"}); err != ErrNotFound {
		t.Errorf("expeted standard error resolving nonexistent ref: %q, got: %q", ErrNotFound, err)
	}

	resolveMe := Ref{
		Username: "test_peer",
		Name:     "my_ds",
	}

	source, err := m.ResolveRef(ctx, &resolveMe)
	if err != nil {
		t.Error(err)
	}
	// source should be local, return the empty string
	expectSource := ""

	if diff := cmp.Diff(expectSource, source); diff != "" {
		t.Errorf("result source mismatch (-want +got):\n%s", diff)
	}

	expect := Ref{
		Username: "test_peer",
		Name:     "my_ds",
		Path:     "/ipfs/QmeXaMpLe",
		ID:       "myInitID",
	}

	if diff := cmp.Diff(expect, resolveMe); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}

	resolveMe = Ref{
		Username: "me",
		Name:     "my_ds",
	}

	if _, err := m.ResolveRef(ctx, &resolveMe); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(expect, resolveMe); diff != "" {
		t.Errorf("'me' shortcut result mismatch. (-want +got):\n%s", diff)
	}

	resolveMe = Ref{
		Username: "test_peer",
		Name:     "my_ds",
		Path:     "/fsi/ill_provide_the_path_thank_you_very_much",
	}

	expect = Ref{
		Username: "test_peer",
		Name:     "my_ds",
		Path:     "/fsi/ill_provide_the_path_thank_you_very_much",
		ID:       "myInitID",
	}

	if _, err := m.ResolveRef(ctx, &resolveMe); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(expect, resolveMe); diff != "" {
		t.Errorf("provided path result mismatch. (-want +got):\n%s", diff)
	}
}
