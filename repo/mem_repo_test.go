package repo

import (
	"context"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/repo/profile"
)

func TestMemRepoResolveRef(t *testing.T) {
	ctx := context.Background()
	fs, err := muxfs.New(ctx, []qfs.Config{
		{Type: "map"},
		{Type: "mem"},
	})
	if err != nil {
		t.Fatal(err)
	}

	pro, err := profile.NewProfile(config.DefaultProfileForTesting())
	if err != nil {
		t.Fatal(err)
	}

	r, err := NewMemRepo(ctx, pro, fs)
	if err != nil {
		t.Fatalf("error creating repo: %s", err.Error())
	}

	dsrefspec.AssertResolverSpec(t, r, func(ref dsref.Ref, author identity.Author, log *oplog.Log) error {
		return r.Logbook().MergeLog(ctx, author, log)
	})
}
