package dsref_test

import (
	"context"
	"testing"

	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
)

func TestMemResolver(t *testing.T) {
	ctx := context.Background()
	m := dsref.NewMemResolver("test_peer")

	if _, err := (*dsref.MemResolver)(nil).ResolveRef(ctx, nil); err != dsref.ErrNotFound {
		t.Errorf("ResolveRef must be nil-callable. expected: %q, got %v", dsref.ErrNotFound, err)
	}

	dsrefspec.ResolverSpec(t, m, func(ref *dsref.Ref) error {
		vi := dsref.VersionInfo{
			InitID:   ref.InitID,
			Username: ref.Username,
			Name:     ref.Name,
			Path:     ref.Path,
		}

		m.Put(vi)
		return nil
	})
}
