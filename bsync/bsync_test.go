package bsync

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/qri-io/qri/manifest"

	"gx/ipfs/QmPSQnBKM9g7BaUcZCvswUJVscQ1ipjmwxN5PXCjkp9EQ7/go-cid"
	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
	coreiface "gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi/interface"
	files "gx/ipfs/QmZMWMvWMVKCbHetJ4RgndbuEF1io2UpUxwQwtNjtYPzSC/go-ipfs-files"
)

func TestSync(t *testing.T) {
	ctx := context.Background()
	_, a, err := makeAPI(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, b, err := makeAPI(ctx)
	if err != nil {
		t.Fatal(err)
	}

	f := files.NewReaderFile("oh_hey", "oh_hey", ioutil.NopCloser(strings.NewReader("y"+strings.Repeat("o", 35000))), nil)
	path, err := a.Unixfs().Add(ctx, f)
	if err != nil {
		t.Fatal(err)
	}

	aGetter := &manifest.NodeGetter{Dag: a.Dag()}
	mfst, err := manifest.NewManifest(ctx, aGetter, path.Cid())
	if err != nil {
		t.Fatal(err)
	}

	bGetter := &manifest.NodeGetter{Dag: b.Dag()}
	receive, err := NewReceive(ctx, bGetter, b.Block(), mfst)
	if err != nil {
		t.Fatal(err)
	}

	rem := &remote{
		receive: receive,
		lng:     bGetter,
		bapi:    b.Block(),
	}

	send, err := NewSend(ctx, aGetter, mfst, rem)
	if err != nil {
		t.Fatal(err)
	}

	if err := send.Do(); err != nil {
		t.Error(err)
	}

	// b should now be able to generate a manifest
	_, err = manifest.NewManifest(ctx, bGetter, path.Cid())
	if err != nil {
		t.Error(err)
	}
}

type TestNodeGetter struct {
	Nodes []ipld.Node
}

var _ ipld.NodeGetter = (*TestNodeGetter)(nil)

func (ng TestNodeGetter) Get(_ context.Context, id cid.Cid) (ipld.Node, error) {
	for _, node := range ng.Nodes {
		if id.Equals(node.Cid()) {
			return node, nil
		}
	}
	return nil, fmt.Errorf("cid not found: %s", id.String())
}

// GetMany returns a channel of NodeOptions given a set of CIDs.
func (ng TestNodeGetter) GetMany(context.Context, []cid.Cid) <-chan *ipld.NodeOption {
	ch := make(chan *ipld.NodeOption)
	ch <- &ipld.NodeOption{
		Err: fmt.Errorf("doesn't support GetMany"),
	}
	return ch
}

// remote implements the Remote interface on a single receive session at a time
type remote struct {
	receive *Receive
	lng     ipld.NodeGetter
	bapi    coreiface.BlockAPI
}

func (r *remote) ReqSession(mfst *manifest.Manifest) (sid string, diff *manifest.Manifest, err error) {
	ctx := context.Background()
	r.receive, err = NewReceive(ctx, r.lng, r.bapi, mfst)
	if err != nil {
		return
	}
	sid = r.receive.sid
	diff = r.receive.diff
	return
}

func (r *remote) PutBlock(sid, hash string, data []byte) Response {
	return r.receive.ReceiveBlock(hash, bytes.NewReader(data))
}
