package base

import (
	"context"

	"github.com/qri-io/dag"

	"gx/ipfs/QmPSQnBKM9g7BaUcZCvswUJVscQ1ipjmwxN5PXCjkp9EQ7/go-cid"
	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
)

// NewManifest generates a manifest for a given node
func NewManifest(ctx context.Context, ng ipld.NodeGetter, path string) (*dag.Manifest, error) {
	id, err := cid.Parse(path)
	if err != nil {
		return nil, err
	}

	return dag.NewManifest(ctx, ng, id)
}
