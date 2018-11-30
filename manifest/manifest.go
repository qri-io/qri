package manifest

import (
	"bytes"
	"context"

	"github.com/ugorji/go/codec"

	"gx/ipfs/QmPSQnBKM9g7BaUcZCvswUJVscQ1ipjmwxN5PXCjkp9EQ7/go-cid"
	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
)

// Node is a subset of the ipld ipld.Node interface
type Node interface {
	// pulled from blocks.Block format
	Cid() cid.Cid
	// Links is a helper function that returns all links within this object
	Links() []*ipld.Link
	// Size returns the size in bytes of the serialized object
	Size() (uint64, error)
}

// Manifest is a DAG of only block names and links (no content)
// node identifiers are stored in a slice "nodes", all other slices reference
// cids by index positions
type Manifest struct {
	Links    [][2]int         `json:"links"`              // links between nodes
	Sections map[string][]int `json:"sections,omitempty"` // sections are lists of logical sub-DAGs by positions in the nodes list
	Nodes    []string         `json:"nodes"`              // list if CIDS contained in the root dag
	Root     int              `json:"root"`               // index if CID in nodes list this manifest is about. The subject of the manifest
	Sizes    []uint64         `json:"sizes"`              // sizes of nodes in bytes
}

// NewManifest generates a manifest from an ipld node
func NewManifest(ctx context.Context, ng ipld.NodeGetter, id cid.Cid) (*Manifest, error) {
	ms := &mstate{
		ctx:  ctx,
		ng:   ng,
		cids: map[string]int{},
		// by convention root is zero b/c root is first node to be added
		m: &Manifest{},
	}

	node, err := ng.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if _, err := ms.addNode(node); err != nil {
		return nil, err
	}
	return ms.m, nil
}

// MarshalCBOR encodes this manifest as CBOR data
func (m *Manifest) MarshalCBOR() (data []byte, err error) {
	buf := &bytes.Buffer{}
	err = codec.NewEncoder(buf, &codec.CborHandle{}).Encode(m)
	data = buf.Bytes()
	return
}

// UnmarshalCBOR decodes a manifest from a byte slice
func UnmarshalCBOR(data []byte) (m *Manifest, err error) {
	m = &Manifest{}
	err = codec.NewDecoder(bytes.NewReader(data), &codec.CborHandle{}).Decode(m)
	return
}

// mstate is a state machine for generating a manifest
type mstate struct {
	ctx  context.Context
	ng   ipld.NodeGetter
	idx  int
	cids map[string]int // lookup table of already-added cids
	m    *Manifest
}

// addNode places a node in the manifest & state machine, recursively adding linked nodes
// addNode returns early if this node is already added to the manifest
func (ms *mstate) addNode(node Node) (int, error) {
	id := node.Cid().String()

	if idx, ok := ms.cids[id]; ok {
		return idx, nil
	}

	// add the node
	idx := ms.idx
	ms.idx++

	ms.cids[id] = idx
	ms.m.Nodes = append(ms.m.Nodes, id)

	// ignore size errors b/c uint64 has no way to represent
	// errored size state as an int (-1), hopefully implementations default to 0
	// when erroring :/
	size, _ := node.Size()

	ms.m.Sizes = append(ms.m.Sizes, size)

	for _, link := range node.Links() {
		linkNode, err := link.GetNode(ms.ctx, ms.ng)
		if err != nil {
			return -1, err
		}

		nodeIdx, err := ms.addNode(linkNode)
		if err != nil {
			return -1, err
		}

		ms.m.Links = append(ms.m.Links, [2]int{idx, nodeIdx})
	}

	return idx, nil
}
