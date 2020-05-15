package spec

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/multiformats/go-multiaddr"
	"github.com/qri-io/qri/dsref"
)

// PutFunc is required to run the ResolverSpec test, when called the Resolver
// should retain the reference for later retrieval
type PutFunc func(ref *dsref.Ref) error

// ResolverSpec confirms the expected behaviour of a dsref.Resolver Interface
// implementation. In addition to this test passing, implementations MUST be
// nil-callable. Please add a nil-callable test to each implementation suite
func ResolverSpec(t *testing.T, r dsref.Resolver, putFunc PutFunc) {
	t.Run("dsrefResolverInterfaceSpec", func(t *testing.T) {
		ctx := context.Background()
		expect := dsref.Ref{
			InitID:   "myInitID",
			Username: "test_peer",
			Name:     "my_ds",
			Path:     "/ipfs/QmeXaMpLe",
		}

		if err := putFunc(&expect); err != nil {
			t.Fatalf("put ref failed: %s", err)
		}

		if _, err := r.ResolveRef(ctx, &dsref.Ref{Username: "username", Name: "does_not_exist"}); err != dsref.ErrNotFound {
			t.Errorf("expected standard error resolving nonexistent ref: %q, got: %q", dsref.ErrNotFound, err)
		}

		resolveMe := dsref.Ref{
			Username: "test_peer",
			Name:     "my_ds",
		}

		source, err := r.ResolveRef(ctx, &resolveMe)
		if err != nil {
			t.Error(err)
		}
		// source should be local, return the empty string
		expectSource := ""

		if diff := cmp.Diff(expectSource, source); diff != "" {
			t.Errorf("result source mismatch (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(expect, resolveMe); diff != "" {
			t.Errorf("result mismatch. (-want +got):\n%s", diff)
		}

		resolveMe = dsref.Ref{
			Username: "test_peer",
			Name:     "my_ds",
			Path:     "/fsi/ill_provide_the_path_thank_you_very_much",
		}

		expect = dsref.Ref{
			Username: "test_peer",
			Name:     "my_ds",
			Path:     "/fsi/ill_provide_the_path_thank_you_very_much",
			InitID:   expect.InitID,
		}

		addr, err := r.ResolveRef(ctx, &resolveMe)
		if err != nil {
			t.Error(err)
		}

		if addr != "" {
			if _, err := multiaddr.NewMultiaddr(addr); err != nil {
				t.Errorf("non-empty source must be a valid multiaddr.\nmultiaddr parse error: %s", err)
			}
		}

		if diff := cmp.Diff(expect, resolveMe); diff != "" {
			t.Errorf("provided path result mismatch. (-want +got):\n%s", diff)
		}
	})
}

// ErrResolversInconsistent indicates two resolvers honored a
// resolution request, but gave differing responses
var ErrResolversInconsistent = fmt.Errorf("inconsistent resolvers")

// InconsistentResolvers confirms two resolvers have different responses for
// the same reference
// this function will not fail the test on error, only write warnings via t.Log
func InconsistentResolvers(t *testing.T, ref dsref.Ref, a, b dsref.Resolver) error {
	err := ConsistentResolvers(t, ref, a, b)
	if err == nil {
		return fmt.Errorf("resolvers are consistent, expected inconsitency")
	}
	if errors.Is(err, ErrResolversInconsistent) {
		return nil
	}

	return err
}

// ConsistentResolvers checks that a set of resolvers return equivelent
// values for a given reference
// this function will not fail the test on error, only write warnings via t.Log
func ConsistentResolvers(t *testing.T, ref dsref.Ref, resolvers ...dsref.Resolver) error {
	var (
		ctx      = context.Background()
		err      error
		resolved *dsref.Ref
	)

	for i, r := range resolvers {
		got := ref.Copy()
		if _, resolveErr := r.ResolveRef(ctx, &got); resolveErr != nil {
			// only legal error return value is dsref.ErrNotFound
			if resolveErr != dsref.ErrNotFound {
				return fmt.Errorf("unexpected error checking consistency with resolver %d (%v): %w", i, r, resolveErr)
			}

			if err == nil && resolved == nil {
				err = resolveErr
				continue
			} else if resolved != nil {
				return fmt.Errorf("%w: index %d (%v) doesn't have reference that was found elsewhere", ErrResolversInconsistent, i, r)
			}
			// err and resolveErr are both ErrNotFound
			continue
		}

		if resolved == nil {
			resolved = &got
			continue
		} else if resolved.Equals(got) {
			continue
		}

		// some resolvers will return fsi-linked paths, which indicates a local
		// checkout, and is an acceptable deviation of the Path value
		// TODO (b5) - use qfs.PathKind here instead of string prefix check after it has been
		// brought up to speed
		if strings.HasPrefix(resolved.Path, "/fsi/") || strings.HasPrefix(got.Path, "/fsi/") {
			t.Logf("caution: comparing resolvers that mix FSI paths with non-FSI paths for consistency. can't check for total equality")
			left := resolved.Copy()
			left.Path = ""
			right := got.Copy()
			right.Path = ""
			if !left.Equals(right) {
				return fmt.Errorf("%w: index %d (%v): %s != %s", ErrResolversInconsistent, i, r, resolved, got)
			}
			continue
		}

		return fmt.Errorf("%w: index %d (%v): %s != %s", ErrResolversInconsistent, i, r, resolved, got)
	}

	return nil
}
