package spec

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/multiformats/go-multiaddr"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/profile"
)

// PutRefFunc adds a reference to a system that retains references
// PutRefFunc is required to run the ResolverSpec test, when called the Resolver
// should retain the reference for later retrieval by the spec test. PutRefFunc
// also passes the author & oplog that back the reference
type PutRefFunc func(ref dsref.Ref, author *profile.Profile, log *oplog.Log) error

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

	profileID := journal.Owner().ID.Encode()
	initID, log, err := GenerateExampleOplog(ctx, journal, dsname, headPath)
	if err != nil {
		t.Fatal(err)
	}

	expectRef := dsref.Ref{
		InitID:    initID,
		ProfileID: profileID,
		Username:  username,
		Name:      dsname,
		Path:      headPath,
	}

	t.Run("dsrefResolverSpec", func(t *testing.T) {
		if err := putFunc(expectRef, journal.Owner(), log); err != nil {
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

		addr, err := r.ResolveRef(ctx, &resolveMe)
		if err != nil {
			t.Error(err)
		}

		if addr != "" {
			if _, err := multiaddr.NewMultiaddr(addr); err != nil {
				if _, urlParseErr := url.Parse(addr); urlParseErr == nil {
					t.Logf("warning: non-empty source must be a valid multiaddr, but returned a url: %s\nURLS will not be permitted in the future", addr)
				} else {
					t.Errorf("non-empty source must be a valid multiaddr.\nmultiaddr parse error: %s", err)
				}
			}
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
			Username:  username,
			Name:      dsname,
			ProfileID: profileID,
			Path:      "/ill_provide_the_path_thank_you_very_much",
			InitID:    expectRef.InitID,
		}

		addr, err = r.ResolveRef(ctx, &resolveMe)
		if err != nil {
			t.Error(err)
		}

		if addr != "" {
			if _, err := multiaddr.NewMultiaddr(addr); err != nil {
				if _, urlParseErr := url.Parse(addr); urlParseErr == nil {
					t.Logf("warning: non-empty source must be a valid multiaddr, but returned a url: %s\nURLS will not be permitted in the future", addr)
				} else {
					t.Errorf("non-empty source must be a valid multiaddr.\nmultiaddr parse error: %s", err)
				}
			}
		}

		if diff := cmp.Diff(expectRef, resolveMe); diff != "" {
			t.Errorf("provided path result mismatch. (-want +got):\n%s", diff)
		}

		// resolveMe = dsref.Ref{
		// 	Username: username,
		// 	Name:     dsname,
		// 	InitID:   initID,
		// }

		// expectRef = dsref.Ref{
		// 	Username:  username,
		// 	Name:      dsname,
		// 	ProfileID: profileID,
		// 	Path:      headPath,
		// 	InitID:    initID,
		// }

		// _, err = r.ResolveRef(ctx, &resolveMe)
		// if err != nil {
		// 	t.Error(err)
		// }
		// if resolveMe.InitID != expectRef.InitID {
		// 	t.Errorf("providing an InitID result mismatch. want: %q\ngot:  %q", expectRef.InitID, resolveMe.InitID)
		// }
		// if diff := cmp.Diff(expectRef, resolveMe); diff != "" {
		// 	t.Errorf("provided InitID result mismatch. (-want +got):\n%s", diff)
		// }

		// providing just an initID MUST populate the alias (human side) of a
		// reference.
		resolveMe = dsref.Ref{
			InitID: initID,

			// erroneous fields need to be be overwritten
			Username: "no not good",
			Name:     "incorrect",
			Path:     "nope_not_right",
		}

		expectRef = dsref.Ref{
			Username:  username,
			Name:      dsname,
			ProfileID: profileID,
			Path:      headPath,
			InitID:    initID,
		}

		_, err = r.ResolveRef(ctx, &resolveMe)
		if err != nil {
			t.Error(err)
		}
		if resolveMe.InitID != expectRef.InitID {
			t.Errorf("providing InitID-only result mismatch. want: %q\ngot:  %q", expectRef.InitID, resolveMe.InitID)
		}
		if diff := cmp.Diff(expectRef, resolveMe); diff != "" {
			t.Errorf("provided InitID-only result mismatch. (-want +got):\n%s", diff)
		}

		// providing a missing initID MUST return ErrRefNotFound or a wrap thereof
		resolveMe = dsref.Ref{
			InitID: "nope_not_here",
		}

		if _, err = r.ResolveRef(ctx, &resolveMe); !errors.Is(err, dsref.ErrRefNotFound) {
			t.Errorf("resolving a missing initID must return ErrRefNotFound or a wrap thereof.\ngot: %s", err)
		}

		// TODO(b5): need to add a test that confirms ResolveRef CANNOT return
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
	t.Helper()

	ms := qfs.NewMemFS()
	pk := testkeys.GetKeyData(9).PrivKey
	pro, err := profile.NewSparsePKProfile(username, pk)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := logbook.NewJournal(*pro, event.NilBus, ms, "/mem/logbook.qfb")
	if err != nil {
		t.Fatal(err)
	}

	return journal
}

// GenerateExampleOplog makes an example dataset history on a given journal,
// returning the initID and a signed log
func GenerateExampleOplog(ctx context.Context, journal *logbook.Book, dsname, headPath string) (string, *oplog.Log, error) {
	author := journal.Owner()
	initID, err := journal.WriteDatasetInit(ctx, author, dsname)
	if err != nil {
		return "", nil, err
	}

	err = journal.WriteVersionSave(ctx, author, &dataset.Dataset{
		ID:       initID,
		Peername: author.Peername,
		Name:     dsname,
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Title:     "initial commit",
		},
		Path:         headPath,
		PreviousPath: "",
	}, nil)
	if err != nil {
		return "", nil, err
	}

	lg, err := journal.UserDatasetBranchesLog(ctx, initID)
	if err != nil {
		return "", nil, err
	}
	if err := lg.Sign(author.PrivKey); err != nil {
		return "", nil, err
	}

	return initID, lg, err
}
