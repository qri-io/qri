package base

import (
	"context"

	"github.com/qri-io/dag"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qfs/cafs"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
)

// NewManifest generates a manifest for a given node
func NewManifest(ctx context.Context, ng ipld.NodeGetter, path string) (*dag.Manifest, error) {
	id, err := cid.Parse(path)
	if err != nil {
		return nil, err
	}

	return dag.NewManifest(ctx, ng, id)
}

// NewDAGInfo generates a DAGInfo for a given node. If a label is provided, it gnenerates a sub-DAGInfo at that
func NewDAGInfo(ctx context.Context, store cafs.Filestore, ng ipld.NodeGetter, path, label string) (*dag.Info, error) {
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
		if err := dsfs.DerefDatasetViz(store, ds); err != nil {
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
