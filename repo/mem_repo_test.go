package repo

import (
	"context"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/profile"
)

func TestMemRepoResolveRef(t *testing.T) {
	ctx := context.Background()
	fs, err := muxfs.New(ctx, []qfs.Config{
		{Type: "mem"},
	})
	if err != nil {
		t.Fatal(err)
	}

	pro, err := profile.NewProfile(config.DefaultProfileForTesting())
	if err != nil {
		t.Fatal(err)
	}

	r, err := NewMemRepo(ctx, pro, fs, event.NilBus)
	if err != nil {
		t.Fatalf("error creating repo: %s", err.Error())
	}

	dsrefspec.AssertResolverSpec(t, r, func(ref dsref.Ref, author profile.Author, log *oplog.Log) error {
		return r.Logbook().MergeLog(ctx, author, log)
	})
}
