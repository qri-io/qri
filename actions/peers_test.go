package actions

import (
	"context"
	"testing"

	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/p2p/test"
)

func TestListPeers(t *testing.T) {
	node := newTestNode(t)

	_, err := ListPeers(node, 10, 0, false)
	if err != nil {
		t.Error(err.Error())
	}

	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(p2p.NewTestableQriNode)
	testNodes, err := p2ptest.NewTestNetwork(ctx, factory, 5)
	if err != nil {
		t.Fatal(err.Error())
	}
	if err := p2ptest.ConnectQriPeers(ctx, testNodes); err != nil {
		t.Fatal(err.Error())
	}

	peers, err := ListPeers(testNodes[0].(*p2p.QriNode), 3, 2, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(peers) != 3 {
		t.Errorf("expected 3 peers, got: %d", len(peers))
	}
}

func TestConnectedQriProfiles(t *testing.T) {
	node := newTestNode(t)

	_, err := ConnectedQriProfiles(node, 100)
	if err != nil {
		t.Error(err.Error())
	}
}
