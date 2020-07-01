package spec

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/multiformats/go-multiaddr"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	testPeers "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
)

// PutRefFunc adds a reference to a system that retains references
// PutRefFunc is required to run the ResolverSpec test, when called the Resolver
// should retain the reference for later retrieval by the spec test. PutRefFunc
// also passes the author & oplog that back the reference
type PutRefFunc func(ref dsref.Ref, author identity.Author, log *oplog.Log) error

// AssertResolverSpec confirms the expected behaviour of a dsref.Resolver
// Interface implementation. In addition to this test passing, implementations
// MUST be nil-callable. Please add a nil-callable test for each implementation
func AssertResolverSpec(t *testing.T, r dsref.Resolver, putFunc PutRefFunc) {
	var (
		ctx              = context.Background()
		username, dsname = "resolve_spec_test_peer", "stored_ref_dataset"
		headPath         = "/ipfs/QmeXaMpLe"
		journal          = ForeignLogbook(t, username)
	)

	initID, log, err := GenerateExampleOplog(ctx, journal, dsname, headPath)
	if err != nil {
		t.Fatal(err)
	}

	expectRef := dsref.Ref{
		InitID:   initID,
		Username: username,
		Name:     dsname,
		Path:     headPath,
	}

	t.Run("dsrefResolverSpec", func(t *testing.T) {
		if err := putFunc(expectRef, journal.Author(), log); err != nil {
			t.Fatalf("put ref failed: %s", err)
		}

		_, err := r.ResolveRef(ctx, &dsref.Ref{Username: "username", Name: "does_not_exist"})
		if err == nil {
			t.Errorf("expected error resolving nonexistent reference, got none")
		} else if !errors.Is(err, dsref.ErrRefNotFound) {
			t.Errorf("expected standard error resolving nonexistent ref: %q, got: %q", dsref.ErrRefNotFound, err)
		}

		resolveMe := dsref.Ref{
			Username: username,
			Name:     dsname,
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

		if diff := cmp.Diff(expectRef, resolveMe); diff != "" {
			t.Errorf("result mismatch. (-want +got):\n%s", diff)
		}

		resolveMe = dsref.Ref{
			Username: username,
			Name:     dsname,
			Path:     "/ill_provide_the_path_thank_you_very_much",
		}

		expectRef = dsref.Ref{
			Username: username,
			Name:     dsname,
			Path:     "/ill_provide_the_path_thank_you_very_much",
			InitID:   expectRef.InitID,
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

		if diff := cmp.Diff(expectRef, resolveMe); diff != "" {
			t.Errorf("provided path result mismatch. (-want +got):\n%s", diff)
		}

		// TODO(b5) - need to add a test that confirms ResolveRef CANNOT return
		// paths outside of logbook HEAD. Subsystems that store references to
		// mutable paths (eg: FSI links) cannot be set as reference resolution
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

// ConsistentResolvers checks that a set of resolvers return equivalent values
// for a given reference
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
			// only legal error return value is dsref.ErrRefNotFound
			if resolveErr != dsref.ErrRefNotFound {
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

		return fmt.Errorf("%w: index %d (%v): %s != %s", ErrResolversInconsistent, i, r, resolved, got)
	}

	return nil
}

// ForeignLogbook creates a logbook to use as an external source of oplog data
func ForeignLogbook(t *testing.T, username string) *logbook.Book {
	pk := testPeers.GetTestPeerInfo(9).PrivKey
	ms := qfs.NewMemFS()
	journal, err := logbook.NewJournal(pk, username, ms, "/mem/logbook.qfb")
	if err != nil {
		t.Fatal(err)
	}

	return journal
}

// GenerateExampleOplog makes an example dataset history on a given journal,
// returning the initID and a signed log
func GenerateExampleOplog(ctx context.Context, journal *logbook.Book, dsname, headPath string) (string, *oplog.Log, error) {
	initID, err := journal.WriteDatasetInit(ctx, dsname)
	if err != nil {
		return "", nil, err
	}

	username := journal.AuthorName()
	err = journal.WriteVersionSave(ctx, initID, &dataset.Dataset{
		Peername: username,
		Name:     dsname,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path:         headPath,
		PreviousPath: "",
	})
	if err != nil {
		return "", nil, err
	}

	lg, err := journal.UserDatasetBranchesLog(ctx, initID)
	if err != nil {
		return "", nil, err
	}
	if err := journal.SignLog(lg); err != nil {
		return "", nil, err
	}

	return initID, lg, err
}
