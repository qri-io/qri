package p2p

import (
	"context"
	"fmt"
	"sort"
	"testing"

	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/event"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo/profile"
)

func TestQriIdentityService(t *testing.T) {
	ctx := context.Background()
	// create a network of connected nodes
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, factory)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	// Convert from test nodes to non-test nodes.
	nodes := make([]*QriNode, len(testPeers))
	for i, node := range testPeers {
		nodes[i] = node.(*QriNode)
	}

	// set up bus & listener for `ETP2PQriPeerConnected` events
	// this is how we will be sure that all the nodes have exchanged qri profiles
	// before moving forward
	bus := event.NewBus(ctx)
	numConns := len(nodes)
	qriPeerConnWaitCh := make(chan struct{})

	disconnectedPeers := peer.IDSlice{}
	numDisconnects := len(nodes)
	disconnectsCh := make(chan struct{})

	watchP2PQriPeersConnected := func(_ context.Context, typ event.Type, payload interface{}) error {
		pro, ok := payload.(*profile.Profile)
		if !ok {
			t.Error("payload for event.ETP2PQriPeerConnected not a *profile.Profile as expected")
			return fmt.Errorf("payload for event.ETP2PQriPeerConnected not a *profile.Profile as expected")
		}
		pid := pro.PeerIDs[0]
		if typ == event.ETP2PQriPeerConnected {
			t.Log("Qri Peer Connected: ", pid)
			numConns--
			if numConns == 0 {
				close(qriPeerConnWaitCh)
			}
		}
		if typ == event.ETP2PQriPeerDisconnected {
			t.Log("Qri Peer Disconnected: ", pid)
			for _, id := range disconnectedPeers {
				if pid == id {
					t.Logf("peer %q has already been disconnected", pid)
					return fmt.Errorf("peer %q has already been disconnected", pid)
				}
			}
			numDisconnects--
			if numDisconnects == 0 {
				close(disconnectsCh)
			}
		}
		return nil
	}
	bus.Subscribe(watchP2PQriPeersConnected, event.ETP2PQriPeerConnected, event.ETP2PQriPeerDisconnected)

	// create a new, disconnected node
	testnode, err := p2ptest.NewNodeWithBus(ctx, factory, bus)
	if err != nil {
		t.Error(err.Error())
		return
	}
	node := testnode.(*QriNode)
	t.Log("node id: ", node.host.ID())

	expectedPeers := peer.IDSlice{}
	for _, node := range nodes {
		expectedPeers = append(expectedPeers, node.host.ID())
	}
	sort.Sort(peer.IDSlice(expectedPeers))
	t.Logf("expected nodes: %v", expectedPeers)

	connectedPeers := peer.IDSlice{}
	connectedPeers = node.qis.ConnectedQriPeers()
	if len(connectedPeers) != 0 {
		t.Errorf("expected 0 peers to be connected to the isolated node, but got %d connected qri peers instead", len(connectedPeers))
	}

	allTestPeers := append(testPeers, testnode)
	if err := p2ptest.ConnectNodes(ctx, allTestPeers); err != nil {
		t.Error(err.Error())
		return
	}
	// wait for all nodes to upgrade to qri peers
	<-qriPeerConnWaitCh

	connectedPeers = node.qis.ConnectedQriPeers()
	sort.Sort(peer.IDSlice(connectedPeers))
	// ensure each peer in the expected list shows up in the connected list

	if len(connectedPeers) == 0 {
		t.Errorf("error exchange qri identities: expected number of connected peers to be %d, got %d", len(expectedPeers), len(connectedPeers))
	}

	if len(expectedPeers) != len(connectedPeers) {
		t.Errorf("expected list of connected peers different then the given list of connected peers: \n  expected: %v\n  got: %v", expectedPeers, connectedPeers)
		return
	}
	different := false
	for i, pid := range expectedPeers {
		if pid != connectedPeers[i] {
			different = true
			break
		}
	}
	if different {
		t.Errorf("expected list of connected peers different then the given list of connected peers: \n  expected: %v\n  got: %v", expectedPeers, connectedPeers)
	}

	for _, id := range connectedPeers {
		if protected := node.host.ConnManager().IsProtected(id, qriSupportKey); !protected {
			t.Errorf("expected peer %q to have a protected connection, but it does not", id)
		}
	}
	for _, id := range connectedPeers {
		pro := node.qis.ConnectedPeerProfile(id)
		if pro == nil {
			t.Errorf("expected to have peer %q's qri profile, but wasn't able to recieve it", id)
		}
	}

	for _, node := range nodes {
		node.host.Close()
	}
	<-disconnectsCh
	connectedPeers = node.qis.ConnectedQriPeers()

	if len(connectedPeers) != 0 {
		t.Errorf("error with catching disconnects, expected 0 remaining connections, got %d to peers %v", len(connectedPeers), connectedPeers)
	}
}
