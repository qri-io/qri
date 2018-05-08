package p2p

import (
	"context"
	"sync"
	"testing"
)

func TestAnnounceConnected(t *testing.T) {
	// create a network of connected nodes
	nodes, err := NewTestDirNetwork(context.Background(), t)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if err := connectQriPeerNodes(context.Background(), nodes); err != nil {
		t.Error(err.Error())
		return
	}

	// create a new, disconnected node
	nds, err := NewTestNetwork(context.Background(), t, 1)
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
	}(node)

	// connected that node to only one member of the network
	if err := connectQriPeerNodes(context.Background(), []*QriNode{node, nodes[0]}); err != nil {
		t.Error(err.Error())
		return
	}

	// have that node announce connection
	if err := node.AnnounceConnected(); err != nil {
		t.Error(err.Error())
		return
	}

	wg.Wait()
}
