package actions

import (
	"context"

	"github.com/qri-io/qri/manifest"
	"github.com/qri-io/qri/p2p"

	"gx/ipfs/QmPSQnBKM9g7BaUcZCvswUJVscQ1ipjmwxN5PXCjkp9EQ7/go-cid"
	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
	"gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi"
	coreiface "gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi/interface"
)

// NewManifest generates a manifest for a given node
func NewManifest(node *p2p.QriNode, path string) (*manifest.Manifest, error) {
	ipfsn, err := node.IPFSNode()
	if err != nil {
		return nil, err
	}

	id, err := cid.Parse(path)
	if err != nil {
		return nil, err
	}

	ng := &nodeGetter{dag: coreapi.NewCoreAPI(ipfsn).Dag()}
	return manifest.NewManifest(node.Context(), ng, id)
}

type nodeGetter struct {
	dag coreiface.DagAPI
}

// Get retrieves nodes by CID. Depending on the NodeGetter
// implementation, this may involve fetching the Node from a remote
// machine; consider setting a deadline in the context.
func (ng *nodeGetter) Get(ctx context.Context, id cid.Cid) (ipld.Node, error) {
	path, err := coreiface.ParsePath(id.String())
	if err != nil {
		return nil, err
	}
	return ng.dag.Get(ctx, path)
}

// GetMany returns a channel of NodeOptions given a set of CIDs.
func (ng *nodeGetter) GetMany(ctx context.Context, cids []cid.Cid) <-chan *ipld.NodeOption {
	ch := make(chan *ipld.NodeOption)
	go func() {
		for _, id := range cids {
			n, err := ng.Get(ctx, id)
			ch <- &ipld.NodeOption{Err: err, Node: n}
		}
	}()
	return ch
}
