package bsync

// import (
// 	"context"
// 	"fmt"
// 	"strconv"

// 	"github.com/multiformats/go-multihash"

// 	"gx/ipfs/QmPSQnBKM9g7BaUcZCvswUJVscQ1ipjmwxN5PXCjkp9EQ7/go-cid"
// 	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
// )

// type node struct {
// 	cid   *cid.Cid
// 	size  uint64
// 	links []*node
// }

// func (n node) String() string        { return n.cid.String() }
// func (n node) Cid() cid.Cid          { return *n.cid }
// func (n node) Size() (uint64, error) { return n.size, nil }
// func (n node) Links() (links []*ipld.Link) {
// 	for _, l := range n.links {
// 		links = append(links, &ipld.Link{
// 			Size: l.size,
// 			Cid:  l.Cid(),
// 		})
// 	}
// 	return
// }

// // Not needed for manifest test:
// func (n node) Loggable() map[string]interface{}                        { return nil }
// func (n node) Copy() ipld.Node                                         { return nil }
// func (n node) RawData() []byte                                         { return nil }
// func (n node) Resolve(path []string) (interface{}, []string, error)    { return nil, nil, nil }
// func (n node) ResolveLink(path []string) (*ipld.Link, []string, error) { return nil, nil, nil }
// func (n node) Stat() (*ipld.NodeStat, error)                           { return nil, nil }
// func (n node) Tree(path string, depth int) []string                    { return nil }

// func NewGraph(layers []layer) (list []ipld.Node) {
// 	root := newNode(2 * kb)
// 	list = append(list, root)
// 	insert(root, layers, &list)
// 	return
// }

// func insert(n *node, layers []layer, list *[]ipld.Node) {
// 	if len(layers) > 0 {
// 		for i := 0; i < layers[0].numChildren; i++ {
// 			ch := newNode(layers[0].size)
// 			n.links = append(n.links, ch)
// 			*list = append(*list, ch)
// 			insert(ch, layers[1:], list)
// 		}
// 	}
// }

// // monotonic content counter for unique, consistent cids
// var content = 0

// func newNode(size uint64) *node {
// 	// Create a cid manually by specifying the 'prefix' parameters
// 	pref := cid.Prefix{
// 		Version:  1,
// 		Codec:    cid.Raw,
// 		MhType:   multihash.SHA2_256,
// 		MhLength: -1, // default length
// 	}

// 	// And then feed it some data
// 	c, err := pref.Sum([]byte(strconv.Itoa(content)))
// 	if err != nil {
// 		panic(err)
// 	}

// 	content++
// 	return &node{
// 		cid:  &c,
// 		size: size,
// 	}
// }

// type TestNodeGetter struct {
// 	Nodes []ipld.Node
// }

// var _ ipld.NodeGetter = (*TestNodeGetter)(nil)

// func (ng TestNodeGetter) Get(_ context.Context, id cid.Cid) (ipld.Node, error) {
// 	for _, node := range ng.Nodes {
// 		if id.Equals(node.Cid()) {
// 			return node, nil
// 		}
// 	}
// 	return nil, fmt.Errorf("cid not found: %s", id.String())
// }

// // GetMany returns a channel of NodeOptions given a set of CIDs.
// func (ng TestNodeGetter) GetMany(context.Context, []cid.Cid) <-chan *ipld.NodeOption {
// 	ch := make(chan *ipld.NodeOption)
// 	ch <- &ipld.NodeOption{
// 		Err: fmt.Errorf("doesn't support GetMany"),
// 	}
// 	return ch
// }
