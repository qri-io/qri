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

// this test is the poster child for re-vamping how we do our p2p test networks
func TestConnectedQriProfiles(t *testing.T) {
	t.Skip("TODO (ramfox): test is flakey.  See comments for full details")
	ctx := context.Background()
	// TODO (ramfox): commented out my work on this test to show what attempts
	// have been made and why they aren't currently working
	// In order to make sure that we have all the Qri Profiles of the nodes we
	// have connected to, we need to make sure that the main node has upgraded
	// each node to a qri connection. However, this is not the only way to have
	// all the profiles in the network, because of the way
	// `upgradeToQriConnection` works, after upgrading a connection, the node
	// then requests all the profiles that the connected peer has, and connects
	// to all those peers)
	// unfortunately, this makes the test super non-deterministic, and it passes
	// and fails for different reasons

	// bus := event.NewBus(ctx)

	// // IMPORTANT: we are waiting for 4 instances of the ETP2PQriPeerConnected events
	// // because currently there are 5 nodes in the
	// numConns := 4
	// waitCh := make(chan struct{}, 1)
	// // This actually fails and causes the test to hang because calling `upgradeQriConnection`
	// // can sometimes result in a non-error response, with NO event publication that a connection
	// // was upgraded (because the peer was theoretically already seen)
	// watchP2PQriPeersConnected := func(_ context.Context, t event.Type, payload interface{}) error {
	// 	if t == event.ETP2PQriPeerConnected {
	// 		pro, ok := payload.(*profile.Profile)
	// 		if !ok {
	// 			return fmt.Errorf("payload for event.ETP2PQriPeerConnected not a *profile.Profile as expected")
	// 		}
	// 		fmt.Println("Qri Peer Connected: ", pro.PeerIDs[0])
	// 		numConns--
	// 		if numConns == 0 {
	// 			waitCh <- struct{}{}
	// 		}
	// 	}
	// 	return nil
	// }
	// bus.Subscribe(watchP2PQriPeersConnected, event.ETP2PQriPeerConnected)

	f := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, f)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	nodes := asQriNodes(testPeers)

	// nodes[0].pub = bus

	if err := p2ptest.ConnectNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	// // this can sometimes hang forever
	// <-waitCh
	pros := nodes[0].ConnectedQriProfiles(ctx)
	if len(pros) != len(nodes)-1 {
		t.Log("My ID:", nodes[0].host.ID())
		t.Log("Conns:")
		for _, conn := range nodes[0].host.Network().Conns() {
			t.Log("    ", conn)
		}
		t.Log("Profiles:")
		for _, pro := range pros {
			t.Log("    ", pro)
		}
		t.Errorf("wrong number of connected profiles. expected: %d, got: %d", len(nodes), len(pros))
		return
	}

	for _, pro := range pros {
		if !pro.Online {
			t.Errorf("expected profile %s to have Online == true", pro.Peername)
		}
	}
}
