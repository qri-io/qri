package mock

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-ipfs/core"
	"github.com/qri-io/qri/base/dsfs"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	p2ptest "github.com/qri-io/qri/p2p/test"
)

func TestMockClient(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	c, err := NewClient(tr.Ctx, tr.NodeA, event.NilBus)
	if err != nil {
		t.Fatal(err)
	}

	ref := dsref.MustParse("this/should_get_made_on_the_fly")
	if _, err := c.PullDataset(tr.Ctx, &ref, ""); err != nil {
		t.Error(err)
	}

	resolve := dsref.MustParse("wut/create_me")
	if _, err := c.NewRemoteRefResolver("").ResolveRef(tr.Ctx, &resolve); err != nil {
		t.Error(err)
	}

	ref = dsref.MustParse("wut/create_me")
	if _, err = c.PullDataset(tr.Ctx, &ref, ""); err != nil {
		t.Error(err)
	}
}

type testRunner struct {
	Ctx   context.Context
	NodeA *p2p.QriNode
}

func newTestRunner(t *testing.T) (tr *testRunner, cleanup func()) {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	tr = &testRunner{
		Ctx: ctx,
	}
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Time{} }

	nodes, _, err := p2ptest.MakeIPFSSwarm(tr.Ctx, true, 2)
	if err != nil {
		t.Fatal(err)
	}

	tr.NodeA = qriNode(ctx, t, tr, "A", nodes[0])

	cleanup = func() {
		dsfs.Timestamp = prevTs
		cancel()
	}
	return tr, cleanup
}

func qriNode(ctx context.Context, t *testing.T, tr *testRunner, peername string, node *core.IpfsNode) *p2p.QriNode {
	repo, err := p2ptest.MakeRepoFromIPFSNode(tr.Ctx, node, peername, event.NewBus(ctx))
	if err != nil {
		t.Fatal(err)
	}

	localResolver := dsref.SequentialResolver(repo.Dscache(), repo)
	qriNode, err := p2p.NewQriNode(repo, testcfg.DefaultP2PForTesting(), repo.Bus(), localResolver)
	if err != nil {
		t.Fatal(err)
	}

	return qriNode
}
