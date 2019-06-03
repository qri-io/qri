package actions

import (
	"github.com/qri-io/dag"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"

	ipld "github.com/ipfs/go-ipld-format"
)

// NewManifest generates a manifest for a given node
func NewManifest(node *p2p.QriNode, path string) (*dag.Manifest, error) {
	ng, err := newNodeGetter(node)
	if err != nil {
		return nil, err
	}

	return base.NewManifest(node.Context(), ng, path)
}

// Missing returns a manifest describing blocks that are not in this node for a given manifest
func Missing(node *p2p.QriNode, m *dag.Manifest) (missing *dag.Manifest, err error) {
	ng, err := newNodeGetter(node)
	if err != nil {
		return nil, err
	}

	return dag.Missing(node.Context(), ng, m)
}

// NewDAGInfo generates a DAGInfo for a given node. If a label is given, it will generate a sub-DAGInfo at thea label.
func NewDAGInfo(node *p2p.QriNode, path, label string) (*dag.Info, error) {
	ng, err := newNodeGetter(node)
	if err != nil {
		return nil, err
	}

	return base.NewDAGInfo(node.Context(), node.Repo.Store(), ng, path, label)
}

// newNodeGetter generates an ipld.NodeGetter from a QriNode
func newNodeGetter(node *p2p.QriNode) (ipld.NodeGetter, error) {
	capi, err := node.IPFSCoreAPI()
	if err != nil {
		return nil, err
	}
	return dag.NewNodeGetter(capi.Dag()), nil
}
