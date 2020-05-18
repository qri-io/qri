package spec

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
)

// PutFunc is required to run the ResolverSpec test, when called the Resolver
// should retain the reference for later retrieval by the spec test. PutFunc
// also passes the author & oplog that back the reference
type PutFunc func(ref dsref.Ref, author identity.Author, log *oplog.Log) error

// ResolverSpec confirms the expected behaviour of a dsref.Resolver Interface
// implementation. In addition to this test passing, implementations MUST be
// nil-callable. Please add a nil-callable test to each implementation suite
func ResolverSpec(t *testing.T, r dsref.Resolver, putFunc PutFunc) {
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

	expect := dsref.Ref{
		InitID:   initID,
		Username: username,
		Name:     dsname,
		Path:     headPath,
	}

	t.Run("dsrefResolverSpec", func(t *testing.T) {
		if err := putFunc(expect, journal.Author(), log); err != nil {
			t.Fatalf("put ref failed: %s", err)
		}

		_, err := r.ResolveRef(ctx, &dsref.Ref{Username: "username", Name: "does_not_exist"})
		if err == nil {
			t.Errorf("expected error resolving nonexistent reference, got none")
		} else if !errors.Is(err, dsref.ErrNotFound) {
			t.Errorf("expected standard error resolving nonexistent ref: %q, got: %q", dsref.ErrNotFound, err)
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

		if diff := cmp.Diff(expect, resolveMe); diff != "" {
			t.Errorf("result mismatch. (-want +got):\n%s", diff)
		}

		resolveMe = dsref.Ref{
			Username: username,
			Name:     dsname,
			Path:     "/ill_provide_the_path_thank_you_very_much",
		}

		expect = dsref.Ref{
			Username: username,
			Name:     dsname,
			Path:     "/ill_provide_the_path_thank_you_very_much",
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

		return fmt.Errorf("%w: index %d (%v): %s != %s", ErrResolversInconsistent, i, r, resolved, got)
	}

	return nil
}

// testAuthorPrivKey is the author of datasets implementers are expected to
// store
func testAuthorPrivKey(t *testing.T) crypto.PrivKey {
	// id: "QmTqawxrPeTRUKS4GSUURaC16o4etPSJv7Akq6a9xqGZUh"
	testPk := `CAASpwkwggSjAgEAAoIBAQDACiqtbAeIR0gKZZfWuNgDssXnQnEQNrAlISlNMrtULuCtsLBk2tZ4C508T4/JQHfvbazZ/aPvkhr9KBaH8AzDU3FngHQnWblGtfm/0FAXbXPfn6DZ1rbA9rx9XpVZ+pUBDve0YxTSPOo5wOOR9u30JEvO47n1R/bF+wtMRHvDyRuoy4H86XxwMR76LYbgSlJm6SSKnrAVoWR9zqjXdaF1QljO77VbivnR5aS9vQ5Sd1mktwgb3SYUMlEGedtcMdLd3MPVCLFzq6cdjhSwVAxZ3RowR2m0hSEE/p6CKH9xz4wkMmjVrADfQTYU7spym1NBaNCrW1f+r4ScDEqI1yynAgMBAAECggEAWuJ04C5IQk654XHDMnO4h8eLsa7YI3w+UNQo38gqr+SfoJQGZzTKW3XjrC9bNTu1hzK4o1JOy4qyCy11vE/3Olm7SeiZECZ+cOCemhDUVsIOHL9HONFNHHWpLwwcUsEs05tpz400xWrezwZirSnX47tpxTgxQcwVFg2Bg07F5BntepqX+Ns7s2XTEc7YO8o77viYbpfPSjrsToahWP7ngIL4ymDjrZjgWTPZC7AzobDbhjTh5XuVKh60eUz0O7/Ezj2QK00NNkkD7nplU0tojZF10qXKCbECPn3pocVPAetTkwB1Zabq2tC2Y10dYlef0B2fkktJ4PAJyMszx4toQQKBgQD+69aoMf3Wcbw1Z1e9IcOutArrnSi9N0lVD7X2B6HHQGbHkuVyEXR10/8u4HVtbM850ZQjVnSTa4i9XJAy98FWwNS4zFh3OWVhgp/hXIetIlZF72GEi/yVFBhFMcKvXEpO/orEXMOJRdLb/7kNpMvl4MQ/fGWOmQ3InkKxLZFJ+wKBgQDA2jUTvSjjFVtOJBYVuTkfO1DKRGu7QQqNeF978ZEoU0b887kPu2yzx9pK0PzjPffpfUsa9myDSu7rncBX1FP0gNmSIAUja2pwMvJDU2VmE3Ua30Z1gVG1enCdl5ZWufum8Q+0AUqVkBdhPxw+XDJStA95FUArJzeZ2MTwbZH0RQKBgDG188og1Ys36qfPW0C6kNpEqcyAfS1I1rgLtEQiAN5GJMTOVIgF91vy11Rg2QVZrp9ryyOI/HqzAZtLraMCxWURfWn8D1RQkQCO5HaiAKM2ivRgVffvBHZd0M3NglWH/cWhxZW9MTRXtWLJX2DVvh0504s9yuAf4Jw6oG7EoAx5AoGBAJluAURO/jSMTTQB6cAmuJdsbX4+qSc1O9wJpI3LRp06hAPDM7ycdIMjwTw8wLVaG96bXCF7ZCGggCzcOKanupOP34kuCGiBkRDqt2tw8f8gA875S+k4lXU4kFgQvf8JwHi02LVxQZF0LeWkfCfw2eiKcLT4fzDV5ppzp1tREQmxAoGAGOXFomnMU9WPxJp6oaw1ZimtOXcAGHzKwiwQiw7AAWbQ+8RrL2AQHq0hD0eeacOAKsh89OXsdS9iW9GQ1mFR3FA7Kp5srjCMKNMgNSBNIb49iiG9O6P6UcO+RbYGg3CkSTG33W8l2pFIjBrtGktF5GoJudAPR4RXhVsRYZMiGag=`
	data, err := base64.StdEncoding.DecodeString(testPk)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := crypto.UnmarshalPrivateKey(data)
	if err != nil {
		t.Fatalf("error unmarshaling private key: %s", err.Error())
	}
	return pk
}

// ForeignLogbook creates a logbook to use as an external source of oplog data
func ForeignLogbook(t *testing.T, username string) *logbook.Book {
	pk := testAuthorPrivKey(t)
	ms := qfs.NewMemFS()
	journal, err := logbook.NewJournal(pk, username, ms, "/mem/logset")
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

	// TODO (b5) - we need UserDatasetRef here b/c it returns the full hierarchy
	// of oplogs. This method should take an InitID
	lg, err := journal.UserDatasetRef(ctx, dsref.Ref{Username: username, Name: dsname})
	if err != nil {
		return "", nil, err
	}
	if err := journal.SignLog(lg); err != nil {
		return "", nil, err
	}

	return initID, lg, err
}
