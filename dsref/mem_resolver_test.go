package dsref_test

import (
	"context"
	"testing"

	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/profile"
)

func TestMemResolver(t *testing.T) {
	ctx := context.Background()
	m := dsref.NewMemResolver("test_peer_mem_resolver")

	if _, err := (*dsref.MemResolver)(nil).ResolveRef(ctx, nil); err != dsref.ErrRefNotFound {
		t.Errorf("ResolveRef must be nil-callable. expected: %q, got %v", dsref.ErrRefNotFound, err)
	}

	dsrefspec.AssertResolverSpec(t, m, func(ref dsref.Ref, author profile.Author, log *oplog.Log) error {
		pid, err := key.IDFromPubKey(author.AuthorPubKey())
		if err != nil {
			return err
		}

		m.Put(dsref.VersionInfo{
			InitID:    ref.InitID,
			ProfileID: pid,
			Username:  ref.Username,
			Name:      ref.Name,
			Path:      ref.Path,
		})
		return nil
	})
}
