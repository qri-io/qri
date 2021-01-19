package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"

	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook/oplog"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/profile"
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
	defer func() {
		for _, node := range nodes {
			node.GoOffline()
		}
	}()

	nodeWithRef := nodes[numNodes-1]

	// set up bus & listener for `ETP2PQriPeerConnected` events
	// this is how we will be sure that all the nodes have exchanged qri profiles
	// before moving forward
	bus := event.NewBus(ctx)
	qriPeerConnWaitCh := make(chan struct{})
	connectedPeersMu := sync.Mutex{}
	connectedPeers := peer.IDSlice{}

	node := &QriNode{}

	watchP2PQriEvents := func(_ context.Context, e event.Event) error {
		pro, ok := e.Payload.(*profile.Profile)
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
			t.Logf("peer %q event occurred, but not an expected peer", pid)
			return nil
		}
		if e.Topic == event.ETP2PQriPeerConnected {
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
	bus.SubscribeTopics(watchP2PQriEvents, event.ETP2PQriPeerConnected)

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

	p2pRefResolver := node.NewP2PRefResolver()

	dsrefspec.AssertResolverSpec(t, p2pRefResolver, func(r dsref.Ref, author profile.Author, _ *oplog.Log) error {
		builder := dscache.NewBuilder()
		pid, err := profile.KeyIDFromPub(author.AuthorPubKey())
		builder.AddUser(r.Username, pid)
		if err != nil {
			return err
		}
		builder.AddDsVersionInfo(dsref.VersionInfo{
			Username:  r.Username,
			InitID:    r.InitID,
			Path:      r.Path,
			ProfileID: pid,
			Name:      r.Name,
		})
		cache := builder.Build()
		nodeWithRef.Repo.Dscache().Assign(cache)
		testResolveRef := &dsref.Ref{
			Username: r.Username,
			Name:     r.Name,
		}
		nodeWithRef.Repo.Dscache().ResolveRef(ctx, testResolveRef)
		return nil
	})
}
