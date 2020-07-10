package p2p

import (
	"context"
	"testing"

	p2ptest "github.com/qri-io/qri/p2p/test"
)

// Convert from test nodes to non-test nodes.
func asQriNodes(testPeers []p2ptest.TestablePeerNode) []*QriNode {
	// Convert from test nodes to non-test nodes.
	peers := make([]*QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*QriNode)
	}
	return peers
}

func TestConnectedQriProfiles(t *testing.T) {
	ctx := context.Background()
	f := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, f)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	nodes := asQriNodes(testPeers)

	pros := nodes[0].ConnectedQriProfiles()
	if len(pros) != len(nodes) {
		t.Log(nodes[0].host.Network().Conns())
		t.Log(pros)
		t.Errorf("wrong number of connected profiles. expected: %d, got: %d", len(nodes), len(pros))
		return
	}

	for _, pro := range pros {
		if !pro.Online {
			t.Errorf("expected profile %s to have Online == true", pro.Peername)
		}
	}
}
