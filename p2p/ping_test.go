package p2p

import (
	"context"
	"testing"

	"github.com/qri-io/qri/p2p/test"
)

func TestPing(t *testing.T) {
	ctx := context.Background()
	f := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, f, 3)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	// Convert from test nodes to non-test nodes.
	peers := asQriNodes(testPeers)

	for i, p1 := range peers {
		for _, p2 := range peers[i+1:] {
			lat, err := p1.Ping(ctx, p2.ID)
			if err != nil {
				t.Errorf("%s -> %s error: %s", p1.ID.Pretty(), p2.ID.Pretty(), err.Error())
				return
			}
			t.Logf("%s Ping: %s: %s", p1.ID, p2.ID, lat)
		}
	}
}
