package manifest

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/multiformats/go-multihash"
	"github.com/ugorji/go/codec"

	"gx/ipfs/QmPSQnBKM9g7BaUcZCvswUJVscQ1ipjmwxN5PXCjkp9EQ7/go-cid"
	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
)

func TestGraphManifestSizeRato(t *testing.T) {
	g := NewGraph([]layer{
		{8, 4 * kb},
		{8, 256 * kb},
		{100, 256 * kb},
	})

	ng := TestingNodeGetter{g}
	mf, err := NewManifest(context.Background(), ng, g[0].Cid())
	if err != nil {
		t.Error(err.Error())
	}

	buf := &bytes.Buffer{}
	enc := codec.NewEncoder(buf, &codec.CborHandle{})
	if err := enc.Encode(mf); err != nil {
		t.Fatal(err.Error())
	}

	size := uint64(0)
	for _, n := range g {
		s, _ := n.Size()
		size += s
	}

	t.Logf("manifest representing %d nodes and %s of content is %s as CBOR", len(mf.Nodes), fileSize(size), fileSize(buf.Len()))
}

/*
		A
	 / \
	B   C
		 / \
		D   E
	 /
	F
*/
func TestNewManifest(t *testing.T) {
	content = 0

	a := newNode(10) // zb2rhd6jTUt94FLVLjrCJ6Wy3NMDxm2sDuwArDfuDaNeHGRi8
	b := newNode(20) // zb2rhdt1wgqfpzMgYf7mefxCWToqUTTyriWA1ctNxmy5WojSz
	c := newNode(30) // zb2rhkwbf5N999rJcRX3D89PVDibZXnctArZFkap4CB36QcAQ
	d := newNode(40) // zb2rhbtsQanqdtuvSceyeKUcT4ao1ge7HULRuRbueGjznWsDP
	e := newNode(50) // zb2rhbhaFdd82br6cP9uUjxQxUyrMFwR3K6uYt6YvUxJtgpSV
	f := newNode(60) // zb2rhnjvVfrzHtyeBcrCt3QUshMoYvEaxPXDykT4MyWvTCKV6
	a.links = []*node{b, c}
	c.links = []*node{d, e}
	d.links = []*node{f}

	ctx := context.Background()
	ng := TestingNodeGetter{[]ipld.Node{a, b, c, d, e, f}}
	mf, err := NewManifest(ctx, ng, a.Cid())
	if err != nil {
		t.Fatal(err)
	}

	exp := &Manifest{
		Nodes: []string{
			"zb2rhd6jTUt94FLVLjrCJ6Wy3NMDxm2sDuwArDfuDaNeHGRi8", // a
			"zb2rhkwbf5N999rJcRX3D89PVDibZXnctArZFkap4CB36QcAQ", // c
			"zb2rhbtsQanqdtuvSceyeKUcT4ao1ge7HULRuRbueGjznWsDP", // d
			"zb2rhbhaFdd82br6cP9uUjxQxUyrMFwR3K6uYt6YvUxJtgpSV", // e
			"zb2rhdt1wgqfpzMgYf7mefxCWToqUTTyriWA1ctNxmy5WojSz", // b
			"zb2rhnjvVfrzHtyeBcrCt3QUshMoYvEaxPXDykT4MyWvTCKV6", // f
		},
		Links: [][2]int{
			{0, 1}, {0, 4}, {1, 2}, {1, 3}, {2, 5},
		},
	}

	verifyManifest(t, exp, mf)
}

func TestNewDAGInfo(t *testing.T) {
	content = 0

	a := newNode(10) // zb2rhd6jTUt94FLVLjrCJ6Wy3NMDxm2sDuwArDfuDaNeHGRi8
	b := newNode(20) // zb2rhdt1wgqfpzMgYf7mefxCWToqUTTyriWA1ctNxmy5WojSz
	c := newNode(30) // zb2rhkwbf5N999rJcRX3D89PVDibZXnctArZFkap4CB36QcAQ
	d := newNode(40) // zb2rhbtsQanqdtuvSceyeKUcT4ao1ge7HULRuRbueGjznWsDP
	e := newNode(50) // zb2rhbhaFdd82br6cP9uUjxQxUyrMFwR3K6uYt6YvUxJtgpSV
	f := newNode(60) // zb2rhnjvVfrzHtyeBcrCt3QUshMoYvEaxPXDykT4MyWvTCKV6
	a.links = []*node{b, c}
	c.links = []*node{d, e}
	d.links = []*node{f}

	ctx := context.Background()
	ng := TestingNodeGetter{[]ipld.Node{a, b, c, d, e, f}}
	di, err := NewDAGInfo(ctx, ng, a.Cid())
	if err != nil {
		t.Fatal(err)
	}

	exp := &DAGInfo{
		Manifest: &Manifest{
			Nodes: []string{
				"zb2rhd6jTUt94FLVLjrCJ6Wy3NMDxm2sDuwArDfuDaNeHGRi8", // a
				"zb2rhkwbf5N999rJcRX3D89PVDibZXnctArZFkap4CB36QcAQ", // c
				"zb2rhbtsQanqdtuvSceyeKUcT4ao1ge7HULRuRbueGjznWsDP", // d
				"zb2rhbhaFdd82br6cP9uUjxQxUyrMFwR3K6uYt6YvUxJtgpSV", // e
				"zb2rhdt1wgqfpzMgYf7mefxCWToqUTTyriWA1ctNxmy5WojSz", // b
				"zb2rhnjvVfrzHtyeBcrCt3QUshMoYvEaxPXDykT4MyWvTCKV6", // f
			},
			Links: [][2]int{
				{0, 1}, {0, 4}, {1, 2}, {1, 3}, {2, 5},
			},
		},
		Sizes: []uint64{10, 30, 40, 50, 20, 60},
	}

	verifyManifest(t, exp.Manifest, di.Manifest)

	if len(exp.Sizes) != len(di.Sizes) {
		t.Errorf("sizes length mismatch. expected: %d. got: %d", len(exp.Sizes), len(di.Sizes))
		return
	}

	for i, s := range exp.Sizes {
		if s != di.Sizes[i] {
			t.Errorf("sizes index %d mismatch. expected: %d, got: %d", i, s, di.Sizes[i])
		}
	}
}

func verifyManifest(t *testing.T, exp, got *Manifest) {
	if len(exp.Nodes) != len(got.Nodes) {
		t.Errorf("nodes length mismatch. %d != %d", len(exp.Nodes), len(got.Nodes))
		return
	}

	for i, id := range exp.Nodes {
		if got.Nodes[i] != id {
			t.Errorf("index: %d order mismatch. expected: %s, got: %s", i, id, got.Nodes[i])
		}
	}

	if len(exp.Links) != len(got.Links) {
		t.Errorf("links length mismatch. %d != %d", len(exp.Links), len(got.Links))
		return
	}

	for i, l := range exp.Links {
		g := got.Links[i]
		if l[0] != g[0] || l[1] != g[1] {
			t.Errorf("links %d mismatch. expected: %v, got: %v", i, l, got.Links[i])
			t.Log(got.Links)
		}
	}
}

type layer struct {
	numChildren int
	size        uint64
}

type node struct {
	cid   *cid.Cid
	size  uint64
	links []*node
}

func (n node) String() string        { return n.cid.String() }
func (n node) Cid() cid.Cid          { return *n.cid }
func (n node) Size() (uint64, error) { return n.size, nil }
func (n node) Links() (links []*ipld.Link) {
	for _, l := range n.links {
		links = append(links, &ipld.Link{
			Size: l.size,
			Cid:  l.Cid(),
		})
	}
	return
}

// Not needed for manifest test:
func (n node) Loggable() map[string]interface{}                        { return nil }
func (n node) Copy() ipld.Node                                         { return nil }
func (n node) RawData() []byte                                         { return nil }
func (n node) Resolve(path []string) (interface{}, []string, error)    { return nil, nil, nil }
func (n node) ResolveLink(path []string) (*ipld.Link, []string, error) { return nil, nil, nil }
func (n node) Stat() (*ipld.NodeStat, error)                           { return nil, nil }
func (n node) Tree(path string, depth int) []string                    { return nil }

func NewGraph(layers []layer) (list []ipld.Node) {
	root := newNode(2 * kb)
	list = append(list, root)
	insert(root, layers, &list)
	return
}

func insert(n *node, layers []layer, list *[]ipld.Node) {
	if len(layers) > 0 {
		for i := 0; i < layers[0].numChildren; i++ {
			ch := newNode(layers[0].size)
			n.links = append(n.links, ch)
			*list = append(*list, ch)
			insert(ch, layers[1:], list)
		}
	}
}

// monotonic content counter for unique, consistent cids
var content = 0

func newNode(size uint64) *node {
	// Create a cid manually by specifying the 'prefix' parameters
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   multihash.SHA2_256,
		MhLength: -1, // default length
	}

	// And then feed it some data
	c, err := pref.Sum([]byte(strconv.Itoa(content)))
	if err != nil {
		panic(err)
	}

	content++
	return &node{
		cid:  &c,
		size: size,
	}
}

type TestingNodeGetter struct {
	Nodes []ipld.Node
}

var _ ipld.NodeGetter = (*TestingNodeGetter)(nil)

func (ng TestingNodeGetter) Get(_ context.Context, id cid.Cid) (ipld.Node, error) {
	for _, node := range ng.Nodes {
		if id.Equals(node.Cid()) {
			return node, nil
		}
	}
	return nil, fmt.Errorf("cid not found: %s", id.String())
}

// GetMany returns a channel of NodeOptions given a set of CIDs.
func (ng TestingNodeGetter) GetMany(context.Context, []cid.Cid) <-chan *ipld.NodeOption {
	ch := make(chan *ipld.NodeOption)
	ch <- &ipld.NodeOption{
		Err: fmt.Errorf("doesn't support GetMany"),
	}
	return ch
}

const (
	kb = 1000
	mb = kb * 1000
	gb = mb * 1000
	tb = gb * 1000
	pb = tb * 1000
)

type fileSize uint64

func (f fileSize) String() string {
	if f < kb {
		return fmt.Sprintf("%d bytes", f)
	} else if f < mb {
		return fmt.Sprintf("%fkb", float32(f)/float32(kb))
	} else if f < gb {
		return fmt.Sprintf("%fMB", float32(f)/float32(mb))
	} else if f < tb {
		return fmt.Sprintf("%fGb", float32(f)/float32(gb))
	} else if f < pb {
		return fmt.Sprintf("%fTb", float32(f)/float32(tb))
	}
	return "NaN"
}

func TestCompletion(t *testing.T) {
	a := Completion{1, 2, 3, 4, 5, 6}
	if a.CompletedBlocks() != 0 {
		t.Errorf("expected completed blocks to equal 0. got: %d", a.CompletedBlocks())
	}

	b := Completion{0, 100}
	if b.CompletedBlocks() != 1 {
		t.Errorf("expected CompletedBlocks == 1. got: %d", b.CompletedBlocks())
	}

	half := Completion{50, 50, 50}
	if half.Percentage() != float32(0.50) {
		t.Errorf("expected half completion to equal 0.5, got: %f", half.Percentage())
	}
	if half.Complete() {
		t.Error("expected unfinished completion to not equal complete")
	}

	done := Completion{100, 100}
	if !done.Complete() {
		t.Error("expected done to equal complete")
	}
}

func TestNewCompletion(t *testing.T) {
	mfst := &Manifest{
		Nodes: []string{"a", "b", "c", "d"},
	}
	missing := &Manifest{
		Nodes: []string{"b", "c"},
	}
	comp := NewCompletion(mfst, missing)
	if comp.Percentage() != 0.5 {
		t.Errorf("expected completion percentage to equal 0.5. got: %f", comp.Percentage())
	}
}
