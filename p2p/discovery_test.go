package p2p

import (
	"context"
	// "sync"
	"fmt"
	"testing"

	// "github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/p2p/test"
	// "github.com/qri-io/qri/repo"
)

func TestRequestStartDiscovery(t *testing.T) {
	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, factory)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	// Convert from test nodes to non-test nodes.
	peers := make([]*QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*QriNode)
	}

	peer0 := peers[0]

	for _, peer := range peers[1:] {
		fmt.Println(peer.SimplePeerInfo())
		peer0.HandlePeerFound(peer.SimplePeerInfo())
	}

	fmt.Println("Addrs:")
	for _, addr := range peer0.Peers() {
		fmt.Printf("\t%s\n", addr)
	}
	fmt.Println("Conns:")
	for _, conn := range peer0.ConnectedPeers() {
		fmt.Printf("\t%s\n", conn)
	}

}
