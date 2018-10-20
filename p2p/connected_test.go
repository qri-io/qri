package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"

	p2ptest "github.com/qri-io/qri/p2p/test"
)

func TestAnnounceConnected(t *testing.T) {
	ctx := context.Background()
	// create a network of connected nodes
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testNodes, err := p2ptest.NewTestDirNetwork(ctx, factory)
	if err != nil {
		t.Error(err.Error())
		return
	}

	// Convert from test nodes to non-test nodes.
	nodes := make([]*QriNode, len(testNodes))
	for i, node := range testNodes {
		nodes[i] = node.(*QriNode)
	}

	for i, a := range nodes {
		for _, b := range nodes[i+1:] {
			bpi := b.SimplePeerInfo()
			a.Host.Connect(ctx, bpi)
			a.HandlePeerFound(bpi)
		}
	}

	// create a new, disconnected node
	nds, err := p2ptest.NewTestNetwork(ctx, factory, 1)
	if err != nil {
		t.Error(err.Error())
		return
	}
	node := nds[0].(*QriNode)
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
	}(node)

	// connected that node to only one member of the network
	npi := nodes[0].SimplePeerInfo()
	node.Host.Connect(ctx, npi)
	node.HandlePeerFound(npi)

	// have that node announce connection
	if err := node.AnnounceConnected(); err != nil {
		t.Error(err.Error())
		return
	}
	wg.Wait()
}
