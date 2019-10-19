package p2p

import (
	"context"
	"testing"

	p2ptest "github.com/qri-io/qri/p2p/test"
)

func TestListPeers(t *testing.T) {
	// node := newTestNode(t)

	// _, err := ListPeers(node, 10, 0, false)
	// if err != nil {
	// 	t.Error(err.Error())
	// }

	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 6)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	peers, err := ListPeers(testPeers[0].(*QriNode), 3, 2, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(peers) != 3 {
		t.Errorf("expected 3 peers, got: %d", len(peers))
	}
}
