package p2p

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/qri-io/qri/p2p/test"
)

// This test connects four nodes to each other, then connects a fifth node to
// one of those four nodes.
// Test passes when the fifth node connects to the other three nodes by asking
// it's one connection for the other three peer's profiles
func TestSharePeers(t *testing.T) {
	fmt.Println("hallo?")
	ctx := context.Background()
	f := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, f, 5)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}

	single := testPeers[0]
	group := testPeers[1:]

	if err := p2ptest.ConnectQriPeers(ctx, group); err != nil {
		t.Errorf("error connecting peers: %s", err.Error())
	}

	nasma := single.(*QriNode)
	done := make(chan bool)
	deadline := time.NewTimer(time.Second * 2)
	go func() {
		for msg := range nasma.ReceiveMessages() {
			if len(nasma.ConnectedPeers()) == len(group) {
				fmt.Println(msg.Type, len(nasma.ConnectedPeers()))
				done <- true
			}
		}
	}()

	nasma.AddQriPeer(group[0].SimplePeerInfo())

	select {
	case <-done:
		return
	case <-deadline.C:
		t.Errorf("peers took too long to connect")
	}
}
