package manifest

import (
	"context"

	"gx/ipfs/QmPSQnBKM9g7BaUcZCvswUJVscQ1ipjmwxN5PXCjkp9EQ7/go-cid"
	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
	coreiface "gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi/interface"
)

// NodeGetter wraps the go-ipfs DagAPI to satistfy the IPLD NodeGetter interface
type NodeGetter struct {
	Dag coreiface.DagAPI
}

// Get retrieves nodes by CID. Depending on the NodeGetter
// implementation, this may involve fetching the Node from a remote
// machine; consider setting a deadline in the context.
func (ng *NodeGetter) Get(ctx context.Context, id cid.Cid) (ipld.Node, error) {
	path, err := coreiface.ParsePath(id.String())
	if err != nil {
		return nil, err
	}
	return ng.Dag.Get(ctx, path)
}

// GetMany returns a channel of NodeOptions given a set of CIDs.
func (ng *NodeGetter) GetMany(ctx context.Context, cids []cid.Cid) <-chan *ipld.NodeOption {
	ch := make(chan *ipld.NodeOption)
	go func() {
		for _, id := range cids {
			n, err := ng.Get(ctx, id)
			ch <- &ipld.NodeOption{Err: err, Node: n}
		}
	}()
	return ch
}
