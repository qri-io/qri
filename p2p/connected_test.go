package p2p

import (
	"context"
	"sync"
	"testing"

	p2ptest "github.com/qri-io/qri/p2p/test"
)

func TestAnnounceConnected(t *testing.T) {
	ctx := context.Background()
	// create a network of connected nodes
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, factory)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	// Convert from test nodes to non-test nodes.
	nodes := make([]*QriNode, len(testPeers))
	for i, node := range testPeers {
		nodes[i] = node.(*QriNode)
	}

	// create a new, disconnected node
	nds, err := p2ptest.NewTestNetwork(ctx, factory, 1)
	if err != nil {
		t.Error(err.Error())
		return
	}
	node := nds[0]
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
			t.Log(node.ID, msg.ID, msg.Type)
			wg.Done()
			remaining--
			if remaining == 0 {
				break
			}
		}
	}(node.(*QriNode))

	// connected that node to only one member of the network
	if err := p2ptest.ConnectQriNodes(ctx, []p2ptest.TestablePeerNode{node, testPeers[0]}); err != nil {
		t.Error(err.Error())
		return
	}

	// have that node announce connection
	if err := node.(*QriNode).AnnounceConnected(ctx); err != nil {
		t.Error(err.Error())
		return
	}

	wg.Wait()
}
