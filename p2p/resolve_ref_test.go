package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"

	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

func TestResolveRef(t *testing.T) {
	ctx := context.Background()
	// create a network of connected nodes
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, factory)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}

	// Convert from test nodes to non-test nodes.
	nodes := make([]*QriNode, len(testPeers))
	numNodes := len(nodes)
	for i, node := range testPeers {
		nodes[i] = node.(*QriNode)
		// closing the discovery process prevents situations where another peer finds
		// our node in the wild and attempts to connect to it while we are trying
		// to connect to that node at the same time. This causes a dial failure.
		nodes[i].Discovery.Close()
	}

	// ref := &dsref.Ref{}
	nodeWithRef := nodes[numNodes-1]

	repoRefs, err := nodeWithRef.Repo.References(0, 1)
	if err != nil {
		t.Fatalf("error getting ref from nodeWithRef %q: %s", nodeWithRef.ID, err)
	}
	if len(repoRefs) == 0 {
		t.Fatal("expected node to have at least one dataset:")
	}

	expectedRef := reporef.ConvertToDsref(repoRefs[0])

	if _, err := nodeWithRef.Repo.ResolveRef(ctx, &expectedRef); err != nil {
		t.Fatalf("expected to be able to resolve the given ref locally: %s", err)
	}

	t.Logf("expected reference: %v", expectedRef)
	t.Logf("from peer %q", nodeWithRef.host.ID())

	ref := &dsref.Ref{
		Username: expectedRef.Username,
		Name:     expectedRef.Name,
	}

	// set up bus & listener for `ETP2PQriPeerConnected` events
	// this is how we will be sure that all the nodes have exchanged qri profiles
	// before moving forward
	bus := event.NewBus(ctx)
	qriPeerConnWaitCh := make(chan struct{})
	connectedPeersMu := sync.Mutex{}
	connectedPeers := peer.IDSlice{}

	node := &QriNode{}

	watchP2PQriEvents := func(_ context.Context, typ event.Type, payload interface{}) error {
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
		for _, peer := range testPeers {
			if pid == peer.Host().ID() {
				expectedPeer = true
				break
			}
		}
		if !expectedPeer {
			t.Logf("peer %q event occured, but not an expected peer", pid)
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
		return nil
	}
	bus.Subscribe(watchP2PQriEvents, event.ETP2PQriPeerConnected)

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
	t.Log("main test node id: ", node.host.ID())

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

	<-qriPeerConnWaitCh

	refResolver := node.NewP2PRefResolver()

	source, err := refResolver.ResolveRef(ctx, ref)
	if err != nil {
		t.Fatalf("error resolving ref: %s", err)
	}

	if !ref.Equals(expectedRef) {
		t.Errorf("error comparing refs, expected: %v, got %v\n", expectedRef, ref)
	}
	if ref.InitID != expectedRef.InitID {
		t.Errorf("error different initIDs: expected: %s, got: %s", expectedRef.InitID, ref.InitID)
	}
	if ref.Username != expectedRef.Username {
		t.Errorf("error different usernames: expected: %s, got: %s", expectedRef.Username, ref.Username)
	}
	if ref.Name != expectedRef.Name {
		t.Errorf("error different dataset names: expected: %s, got: %s", expectedRef.Name, ref.Name)
	}
	if ref.ProfileID != expectedRef.ProfileID {
		t.Errorf("error different profileIDs: expected: %s, got: %s", expectedRef.ProfileID, ref.ProfileID)
	}

	t.Log("source is: ", source)
}
