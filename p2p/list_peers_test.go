package p2p

import (
	"context"
	"testing"

	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/profile"
)

func TestListPeers(t *testing.T) {
	t.Skip("ramfox: flakey test until p2p test network setup is refactored")
	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 6)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	userID := profile.IDFromPeerID(testPeers[0].SimpleAddrInfo().ID)
	peers, err := ListPeers(testPeers[0].(*QriNode), userID, 2, 3, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(peers) != 3 {
		t.Errorf("expected 3 peers, got: %d", len(peers))
	}
}
