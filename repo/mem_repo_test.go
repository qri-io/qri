package repo

import (
	"context"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/repo/profile"
)

func TestMemRepoResolveRef(t *testing.T) {
	pro, err := profile.NewProfile(config.DefaultProfileForTesting())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	store := cafs.NewMapstore()
	fs := qfs.NewMemFS()
	r, err := NewMemRepo(pro, store, fs, profile.NewMemStore())
	if err != nil {
		t.Fatalf("error creating repo: %s", err.Error())
	}

	dsrefspec.ResolverSpec(t, r, func(ref dsref.Ref, author identity.Author, log *oplog.Log) error {
		return r.Logbook().MergeLog(ctx, author, log)
	})
}
