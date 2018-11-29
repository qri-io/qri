package base

import (
	"context"

	"github.com/qri-io/qri/manifest"

	"gx/ipfs/QmPSQnBKM9g7BaUcZCvswUJVscQ1ipjmwxN5PXCjkp9EQ7/go-cid"
	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
)

// NewManifest generates a manifest for a given node
func NewManifest(ctx context.Context, ng ipld.NodeGetter, path string) (*manifest.Manifest, error) {
	id, err := cid.Parse(path)
	if err != nil {
		return nil, err
	}

	return manifest.NewManifest(ctx, ng, id)
}

// Missing returns a manifest describing blocks that are not in this node for a given manifest
func Missing(ctx context.Context, ng ipld.NodeGetter, m *manifest.Manifest) (missing *manifest.Manifest, err error) {
	var nodes []string

	for _, idstr := range m.Nodes {
		id, err := cid.Parse(idstr)
		if err != nil {
			return nil, err
		}
		if _, err := ng.Get(ctx, id); err == ipld.ErrNotFound {
			nodes = append(nodes, id.String())
		} else if err != nil {
			return nil, err
		}
	}
	return &manifest.Manifest{Nodes: nodes}, nil
}
