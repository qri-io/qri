package p2p

import (
	"context"
	"testing"
	"time"

	p2ptest "github.com/qri-io/qri/p2p/test"
)

// This test connects four nodes to each other, then connects a fifth node to
// one of those four nodes.
// Test passes when the fifth node connects to the other three nodes by asking
// it's one connection for the other three peer's profiles
func TestSharePeers(t *testing.T) {
	ctx := context.Background()
	f := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, f, 5)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}

	single := testPeers[0]
	group := testPeers[1:]

	if err := p2ptest.ConnectQriNodes(ctx, group); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	nasma := single.(*QriNode)
	done := make(chan bool)
	deadline := time.NewTimer(time.Second * 2)

	if err := p2ptest.ConnectQriNodes(ctx, []p2ptest.TestablePeerNode{nasma, group[0]}); err != nil {
		t.Fatalf("error connecting single node to single group node %s", err.Error())
	}

	go func() {
		for range nasma.ReceiveMessages() {
			t.Logf("connected to peer. %d/%d", len(nasma.ConnectedPeers()), len(group))
			if len(nasma.ConnectedPeers()) >= len(group) {
				done <- true
			}
		}
	}()

	select {
	case <-done:
		return
	case <-deadline.C:
		t.Errorf("peers took too long to connect")
	}
}
