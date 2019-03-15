package base

import (
	"context"

	"github.com/qri-io/dag"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qfs/cafs"

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

// NewDAGInfo generates a DAGInfo for a given node
func NewDAGInfo(ctx context.Context, store cafs.Filestore, ng ipld.NodeGetter, path string) (*dag.Info, error) {
	id, err := cid.Parse(path)
	if err != nil {
		return nil, err
	}

	info, err := dag.NewInfo(ctx, ng, id)
	if err != nil {
		return nil, err
	}
	// get referenced version of dataset
	ds, err := dsfs.LoadDatasetRefs(store, path)
	if err != nil {
		return nil, err
	}
	info.Labels = map[string]int{}
	prefix := store.PathPrefix()
	if ds.BodyPath != "" {
		err := info.AddLabelByID("body", dsfs.GetHashBase(ds.BodyPath, prefix))
		if err != nil {
			return nil, err
		}
	}
	if ds.Viz != nil {
		err := info.AddLabelByID("viz", dsfs.GetHashBase(ds.Viz.Path, prefix))
		if err != nil {
			return nil, err
		}
	}
	if ds.Transform != nil {
		err := info.AddLabelByID("transform", dsfs.GetHashBase(ds.Transform.Path, prefix))
		if err != nil {
			return nil, err
		}
	}
	if ds.Meta != nil {
		err := info.AddLabelByID("meta", dsfs.GetHashBase(ds.Meta.Path, prefix))
		if err != nil {
			return nil, err
		}
	}
	if ds.Structure != nil {
		err := info.AddLabelByID("structure", dsfs.GetHashBase(ds.Structure.Path, prefix))
		if err != nil {
			return nil, err
		}
	}
	if ds.Commit != nil {
		err := info.AddLabelByID("commit", dsfs.GetHashBase(ds.Commit.Path, prefix))
		if err != nil {
			return nil, err
		}
	}
	return info, nil
}

// NewSubDAGInfo generates a SubDAGInfo for a given node at a given label
func NewSubDAGInfo(ctx context.Context, store cafs.Filestore, ng ipld.NodeGetter, path, label string) (*dag.Info, error) {
	info, err := NewDAGInfo(ctx, store, ng, path)
	if err != nil {
		return nil, err
	}
	return info.InfoAtLabel(label)
}
