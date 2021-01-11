package p2p

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/event"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/profile"
)

// Hello fellow devs. Let my pain be your comfort and plz use my notes here
// as a guide on how to not pull out your hair when testing the p2p connections
// things to note:
//   - once a libp2p connection has been established, the QriProfileService will
//   check if that remote peers speaks qri. If it does, it will protect that
//   connection, add it to a list of current qri connections, and ask that peer
//   for it's qri profile. It will then emit a `event.ETP2PQriPeerConnected` event
//   - if a libp2p connection has been broken, aka disconnected, the QriProfile-
//   Service will remove that connection from the current qri connections and
//   emit a `event.ETP2PQriPeerDisconnected` event
//   - we can use these events to coordinate on
//   - QriNodes have a Discovery process that the peers use to find each other
//   in the "wild". This is why you may see connections to peers that you haven't
//   explicitly connected to. When this is happening locally, it may happen so
//   quickly that the connections can collide causing errors. Unless you are
//   specifically testing that the peers can find themselves on their own, it
//   may be best to disable Discovery by calling `node.Discovery.Close()`
//   - when coordinating on events, ensure that you are using a mutex lock to
//   prevent any race conditions
//   - use channels to wait for all connections or disconnections to occur. a
//   closed channel never blocks!

func TestQriProfileService(t *testing.T) {
	ctx := context.Background()
	numNodes := 5 // number of nodes we want (besides the main test node) in this test
	// create a network of connected nodes
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, numNodes)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}

	// Convert from test nodes to non-test nodes.
	nodes := make([]*QriNode, len(testPeers))
	for i, node := range testPeers {
		nodes[i] = node.(*QriNode)
		// closing the discovery process prevents situations where another peer finds
		// our node in the wild and attempts to connect to it while we are trying
		// to connect to that node at the same time. This causes a dial failure.
		nodes[i].Discovery.Close()
	}

	expectedPeers := peer.IDSlice{}
	for _, node := range nodes {
		expectedPeers = append(expectedPeers, node.host.ID())
	}
	sort.Sort(peer.IDSlice(expectedPeers))
	t.Logf("expected nodes: %v", expectedPeers)

	// set up bus & listener for `ETP2PQriPeerConnected` events
	// this is how we will be sure that all the nodes have exchanged qri profiles
	// before moving forward
	bus := event.NewBus(ctx)
	qriPeerConnWaitCh := make(chan struct{})
	connectedPeersMu := sync.Mutex{}
	connectedPeers := peer.IDSlice{}

	disconnectedPeers := peer.IDSlice{}
	disconnectedPeersMu := sync.Mutex{}
	disconnectsCh := make(chan struct{})

	node := &QriNode{}

	unexpectedPeers := peer.IDSlice{}

	watchP2PQriEvents := func(_ context.Context, typ event.Type, ts int64, sid string, payload interface{}) error {
		pro, ok := payload.(*profile.Profile)
		if !ok {
			t.Error("payload for event.ETP2PQriPeerConnected not a *profile.Profile as expected")
			return fmt.Errorf("payload for event.ETP2PQriPeerConnected not a *profile.Profile as expected")
		}
		if pro == nil {
			t.Error("error: event payload is a nil profile")
			return fmt.Errorf("err: event payload is a nil profile")
		}
		pid := pro.PeerIDs[0]
		expectedPeer := false
		for _, id := range expectedPeers {
			if pid == id {
				expectedPeer = true
				break
			}
		}
		if !expectedPeer {
			t.Logf("peer %q event occurred, but not an expected peer", pid)
			unexpectedPeers = append(unexpectedPeers, pid)
			return nil
		}
		if typ == event.ETP2PQriPeerConnected {
			connectedPeersMu.Lock()
			defer connectedPeersMu.Unlock()
			t.Log("Qri Peer Connected: ", pid)
			for _, id := range connectedPeers {
				if pid == id {
					t.Logf("peer %q has already been connected", pid)
					return nil
				}
			}
			connectedPeers = append(connectedPeers, pid)
			if len(connectedPeers) == numNodes {
				close(qriPeerConnWaitCh)
			}
		}
		if typ == event.ETP2PQriPeerDisconnected {
			disconnectedPeersMu.Lock()
			defer disconnectedPeersMu.Unlock()
			t.Log("Qri Peer Disconnected: ", pid)
			for _, id := range disconnectedPeers {
				if pid == id {
					t.Logf("peer %q has already been disconnected", pid)
					return nil
				}
			}
			disconnectedPeers = append(disconnectedPeers, pid)
			if len(disconnectedPeers) == numNodes {
				close(disconnectsCh)
			}
		}
		return nil
	}
	bus.Subscribe(watchP2PQriEvents, event.ETP2PQriPeerConnected, event.ETP2PQriPeerDisconnected)

	// create a new, disconnected node
	testnode, err := p2ptest.NewNodeWithBus(ctx, factory, bus)
	if err != nil {
		t.Fatal(err)
	}
	node = testnode.(*QriNode)
	// closing the discovery process prevents situations where another peer finds
	// our node in the wild and attempts to connect to it while we are trying
	// to connect to that node at the same time. This causes a dial failure.
	node.Discovery.Close()
	defer node.GoOffline()
	t.Log("node id: ", node.host.ID())

	connectedPeers = node.qis.ConnectedQriPeers()
	if len(connectedPeers) != 0 {
		t.Errorf("expected 0 peers to be connected to the isolated node, but got %d connected qri peers instead", len(connectedPeers))
	}

	// explicitly connect the main test node to each other node in the network
	for _, groupNode := range nodes {
		addrInfo := peer.AddrInfo{
			ID:    groupNode.Host().ID(),
			Addrs: groupNode.Host().Addrs(),
		}
		err := node.Host().Connect(context.Background(), addrInfo)
		if err != nil {
			t.Logf("error connecting to peer %q: %s", groupNode.Host().ID(), err)
		}
	}

	// wait for all nodes to upgrade to qri peers
	<-qriPeerConnWaitCh

	// get a list of connected peers, according to the QriProfileService
	connectedPeers = node.qis.ConnectedQriPeers()
	sort.Sort(peer.IDSlice(connectedPeers))

	// ensure the lists are the same
	if len(connectedPeers) == 0 {
		t.Errorf("error exchange qri identities: expected number of connected peers to be %d, got %d", len(expectedPeers), len(connectedPeers))
	}

	if len(unexpectedPeers) != 0 {
		t.Errorf("unexpected peers found: %v", unexpectedPeers)
		for _, pid := range unexpectedPeers {
			protocols, err := node.host.Peerstore().GetProtocols(pid)
			if err != nil {
				t.Errorf("error getting peer %q protocols: %s", pid, err)
			} else {
				t.Logf("peer %q speaks protocols: %v", pid, protocols)
			}
		}
	}

	if len(expectedPeers) != len(connectedPeers) {
		t.Errorf("expected list of connected peers different then the given list of connected peers: \n  expected: %v\n  got: %v", expectedPeers, connectedPeers)
		for _, peer := range connectedPeers {
			pro, err := node.Repo.Profiles().PeerProfile(peer)
			if err != nil {
				t.Errorf("error getting peer %q profile: %s", peer, err)
			}
			t.Logf("%s, %v", peer, pro)
		}
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

	// check that each connection is protected, and the we have a profile for each
	for _, id := range connectedPeers {
		if protected := node.host.ConnManager().IsProtected(id, qriSupportKey); !protected {
			t.Errorf("expected peer %q to have a protected connection, but it does not", id)
		}
		pro := node.qis.ConnectedPeerProfile(id)
		if pro == nil {
			t.Errorf("expected to have peer %q's qri profile, but wasn't able to receive it", id)
		}
	}

	// disconnect each node
	for _, node := range nodes {
		node.GoOffline()
	}

	<-disconnectsCh
	// get a list of connected qri peers according to the QriProfileService
	connectedPeers = node.qis.ConnectedQriPeers()

	if len(connectedPeers) != 0 {
		t.Errorf("error with catching disconnects, expected 0 remaining connections, got %d to peers %v", len(connectedPeers), connectedPeers)
	}
}

func TestDiscoveryConnection(t *testing.T) {
	t.Skip("enable this test to see if nodes will find each other on your machine using their Discovery methods without getting orders to explicitily connect")
	ctx := context.Background()
	// create a network of connected nodes
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	// create a new, disconnected node
	busA := event.NewBus(ctx)

	watchP2PQriEvents := func(_ context.Context, typ event.Type, ts int64, sid string, payload interface{}) error {
		pro, ok := payload.(*profile.Profile)
		if !ok {
			t.Error("payload for event.ETP2PQriPeerConnected not a *profile.Profile as expected")
			return fmt.Errorf("payload for event.ETP2PQriPeerConnected not a *profile.Profile as expected")
		}
		pid := pro.PeerIDs[0]
		if typ == event.ETP2PQriPeerConnected {
			t.Log("Qri Peer Connected: ", pid)
		}
		if typ == event.ETP2PQriPeerDisconnected {
			t.Log("Qri Peer Disconnected: ", pid)
		}
		return nil
	}
	busA.Subscribe(watchP2PQriEvents, event.ETP2PQriPeerConnected, event.ETP2PQriPeerDisconnected)

	testNodeA, err := p2ptest.NewNodeWithBus(ctx, factory, busA)
	if err != nil {
		t.Error(err.Error())
		return
	}
	nodeA := testNodeA.(*QriNode)

	testNodeB, err := p2ptest.NewNodeWithBus(ctx, factory, event.NilBus)
	if err != nil {
		t.Error(err.Error())
		return
	}
	nodeB := testNodeB.(*QriNode)

	time.Sleep(time.Second * 30)

	t.Errorf("nodeA's connections: %v", nodeA.ConnectedQriPeerIDs())
	t.Errorf("nodeB's connections: %v", nodeB.ConnectedQriPeerIDs())
	nodeA.GoOffline()
	nodeB.GoOffline()
}
