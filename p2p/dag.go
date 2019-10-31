package p2p

import (
	"context"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/qri-io/dag"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base/dsfs"
)

// NewManifest generates a manifest for a given node
func (node *QriNode) NewManifest(ctx context.Context, path string) (*dag.Manifest, error) {
	ng, err := newNodeGetter(node)
	if err != nil {
		return nil, err
	}

	id, err := cid.Parse(path)
	if err != nil {
		return nil, err
	}

	return dag.NewManifest(ctx, ng, id)
}

// MissingManifest returns a manifest describing blocks that are not in this
// node for a given manifest
func (node *QriNode) MissingManifest(ctx context.Context, m *dag.Manifest) (missing *dag.Manifest, err error) {
	ng, err := newNodeGetter(node)
	if err != nil {
		return nil, err
	}

	return dag.Missing(ctx, ng, m)
}

// NewDAGInfo generates a DAGInfo for a given node. If a label is given, it will generate a sub-DAGInfo at thea label.
func (node *QriNode) NewDAGInfo(ctx context.Context, path, label string) (*dag.Info, error) {
	ng, err := newNodeGetter(node)
	if err != nil {
		return nil, err
	}

	return newDAGInfo(ctx, node.Repo.Store(), ng, path, label)
}

// newDAGInfo generates a DAGInfo for a given node. If a label is provided,
// it generates a sub-DAGInfo at that label
func newDAGInfo(ctx context.Context, store cafs.Filestore, ng ipld.NodeGetter, path, label string) (*dag.Info, error) {
	id, err := cid.Parse(path)
	if err != nil {
		return nil, err
	}

	info, err := dag.NewInfo(ctx, ng, id)
	if err != nil {
		return nil, err
	}
	// get referenced version of dataset
	ds, err := dsfs.LoadDatasetRefs(ctx, store, path)
	if err != nil {
		return nil, err
	}
	info.Labels = map[string]int{}
	prefix := store.PathPrefix()
	if ds.BodyPath != "" {
		err := info.AddLabelByID("bd", dsfs.GetHashBase(ds.BodyPath, prefix))
		if err != nil {
			return nil, err
		}
	}
	if ds.Viz != nil {
		err := info.AddLabelByID("vz", dsfs.GetHashBase(ds.Viz.Path, prefix))
		if err != nil {
			return nil, err
		}
		if err := dsfs.DerefDatasetViz(ctx, store, ds); err != nil {
			return nil, err
		}
		if ds.Viz.RenderedPath != "" {
			err := info.AddLabelByID("rd", dsfs.GetHashBase(ds.Viz.RenderedPath, prefix))
			if err != nil {
				return nil, err
			}
		}
	}
	if ds.Transform != nil {
		err := info.AddLabelByID("tf", dsfs.GetHashBase(ds.Transform.Path, prefix))
		if err != nil {
			return nil, err
		}
	}
	if ds.Meta != nil {
		err := info.AddLabelByID("md", dsfs.GetHashBase(ds.Meta.Path, prefix))
		if err != nil {
			return nil, err
		}
	}
	if ds.Structure != nil {
		err := info.AddLabelByID("st", dsfs.GetHashBase(ds.Structure.Path, prefix))
		if err != nil {
			return nil, err
		}
	}
	if ds.Commit != nil {
		err := info.AddLabelByID("cm", dsfs.GetHashBase(ds.Commit.Path, prefix))
		if err != nil {
			return nil, err
		}
	}

	if label != "" {
		return info.InfoAtLabel(label)
	}
	return info, nil
}

// newNodeGetter generates an ipld.NodeGetter from a QriNode
func newNodeGetter(node *QriNode) (ipld.NodeGetter, error) {
	capi, err := node.IPFSCoreAPI()
	if err != nil {
		return nil, err
	}
	return dag.NewNodeGetter(capi.Dag()), nil
}
