package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/qri-io/qri/event"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo/profile"
)

func TestAnnounceConnected(t *testing.T) {
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
	numConnsMu := sync.Mutex{}
	qriPeerConnWaitCh := make(chan struct{})
	watchP2PQriPeersConnected := func(_ context.Context, typ event.Type, payload interface{}) error {
		numConnsMu.Lock()
		defer numConnsMu.Unlock()
		if typ == event.ETP2PQriPeerConnected {
			pro, ok := payload.(*profile.Profile)
			if !ok {
				return fmt.Errorf("payload for event.ETP2PQriPeerConnected not a *profile.Profile as expected")
			}
			t.Log("Qri Peer Connected: ", pro.PeerIDs[0])
			numConns--
			if numConns == 0 {
				close(qriPeerConnWaitCh)
			}
		}
		return nil
	}
	bus.Subscribe(watchP2PQriPeersConnected, event.ETP2PQriPeerConnected)

	// create a new, disconnected node
	testnode, err := p2ptest.NewNodeWithBus(ctx, factory, bus)
	if err != nil {
		t.Error(err.Error())
		return
	}
	node := testnode.(*QriNode)

	wg := sync.WaitGroup{}
	// TODO - this logic needs some precise-ifying to make this test more robust
	// Basically we're firing too many messages, causing negative waits. We should
	// sort out which messages we really care about & make this test check in a more
	// exact manner
	wg.Add(len(nodes))
	remaining := len(nodes)
	go func(node *QriNode) {

		r := node.ReceiveMessages()
		for {
			msg := <-r
			t.Logf("%s, %s, %s", node.ID, msg.ID, msg.Type)
			wg.Done()
			remaining--
			if remaining == 0 {
				break
			}
		}
	}(node)

	// connected that node to only one member of the network
	if err := p2ptest.ConnectNodes(ctx, []p2ptest.TestablePeerNode{node, testPeers[0]}); err != nil {
		t.Error(err.Error())
		return
	}
	// wait for all nodes to upgrade to qri peers
	<-qriPeerConnWaitCh
	// have that node announce connection
	if err := node.AnnounceConnected(ctx); err != nil {
		t.Error(err.Error())
		return
	}
	wg.Wait()
}
