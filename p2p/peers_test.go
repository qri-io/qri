package p2p

import (
	"context"
	"testing"
)

func TestConnectedQriProfiles(t *testing.T) {
	nodes, err := NewTestDirNetwork(context.Background(), t)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if err := connectQriPeerNodes(context.Background(), nodes); err != nil {
		t.Error(err.Error())
		return
	}

	pros := nodes[0].ConnectedQriProfiles()
	if len(pros) != len(nodes)-1 {
		t.Log(nodes[0].Host.Network().Conns())
		t.Log(pros)
		t.Errorf("wrong number of connected profiles. expected: %d, got: %d", len(nodes)-1, len(pros))
		return
	}

	for _, pro := range pros {
		if !pro.Online {
			t.Errorf("expected profile %s to have Online == true", pro.Peername)
		}
	}
}
