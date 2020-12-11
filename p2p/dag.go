package p2p

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/qri-io/dag"
	"github.com/qri-io/qfs"
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

	return newDAGInfo(ctx, node.Repo.Filesystem(), ng, path, label)
}

// newDAGInfo generates a DAGInfo for a given node. If a label is provided,
// it generates a sub-DAGInfo at that label
func newDAGInfo(ctx context.Context, fs qfs.Filesystem, ng ipld.NodeGetter, path, label string) (*dag.Info, error) {
	id, err := cid.Parse(path)
	if err != nil {
		return nil, err
	}

	info, err := dag.NewInfo(ctx, ng, id)
	if err != nil {
		return nil, err
	}
	// get referenced version of dataset
	ds, err := dsfs.LoadDatasetRefs(ctx, fs, path)
	if err != nil {
		return nil, err
	}

	info.Labels = map[string]int{}
	if ds.BodyPath != "" {
		cid, err := dsfs.GetHashCID(ds.BodyPath)
		if err != nil {
			return nil, err
		}
		if err = info.AddLabelByID("bd", cid); err != nil {
			return nil, fmt.Errorf("adding body label: %w", err)
		}
	}
	if ds.Viz != nil && ds.Viz.Path != "" {
		cid, err := dsfs.GetHashCID(ds.Viz.Path)
		if err != nil {
			return nil, err
		}
		if err = info.AddLabelByID("vz", cid); err != nil {
			return nil, fmt.Errorf("adding viz label: %w", err)
		}
		if err := dsfs.DerefViz(ctx, fs, ds); err != nil {
			return nil, err
		}
		if ds.Viz.RenderedPath != "" {
			cid, err := dsfs.GetHashCID(ds.Viz.RenderedPath)
			if err != nil {
				return nil, err
			}
			if err = info.AddLabelByID("rd", cid); err != nil {
				return nil, err
			}
		}
	}
	if ds.Transform != nil && ds.Transform.Path != "" {
		cid, err := dsfs.GetHashCID(ds.Transform.Path)
		if err != nil {
			return nil, err
		}
		if err = info.AddLabelByID("tf", cid); err != nil {
			return nil, fmt.Errorf("adding transform label: %w", err)
		}
	}
	if ds.Meta != nil && ds.Meta.Path != "" {
		cid, err := dsfs.GetHashCID(ds.Meta.Path)
		if err != nil {
			return nil, err
		}
		if err = info.AddLabelByID("md", cid); err != nil {
			return nil, fmt.Errorf("adding meta label %w", err)
		}
	}
	if ds.Structure != nil && ds.Structure.Path != "" {
		cid, err := dsfs.GetHashCID(ds.Structure.Path)
		if err != nil {
			return nil, err
		}
		if err = info.AddLabelByID("st", cid); err != nil {
			return nil, fmt.Errorf("adding structure label: %w", err)
		}
	}
	if ds.Stats != nil && ds.Stats.Path != "" {
		cid, err := dsfs.GetHashCID(ds.Stats.Path)
		if err != nil {
			return nil, err
		}

		if err := info.AddLabelByID("sa", cid); err != nil {
			return nil, fmt.Errorf("adding stats label: %w", err)
		}
	}
	if ds.Commit != nil && ds.Commit.Path != "" {
		cid, err := dsfs.GetHashCID(ds.Commit.Path)
		if err != nil {
			return nil, err
		}

		if err = info.AddLabelByID("cm", cid); err != nil {
			return nil, fmt.Errorf("adding commit label: %w", err)
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
