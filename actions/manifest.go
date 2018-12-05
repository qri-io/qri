package actions

import (
	"github.com/qri-io/dag"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"

	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
	"gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi"
	coreiface "gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi/interface"
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

// newNodeGetter generates an ipld.NodeGetter from a QriNode
func newNodeGetter(node *p2p.QriNode) (ng ipld.NodeGetter, err error) {
	ipfsn, err := node.IPFSNode()
	if err != nil {
		return nil, err
	}

	ng = &dag.NodeGetter{Dag: coreapi.NewCoreAPI(ipfsn).Dag()}
	return
}

// newIPFSCoreAPI generates an ipld.NodeGetter from a QriNode
func newIPFSCoreAPI(node *p2p.QriNode) (capi coreiface.CoreAPI, err error) {
	ipfsn, err := node.IPFSNode()
	if err != nil {
		return nil, err
	}

	return coreapi.NewCoreAPI(ipfsn), nil
}
