package p2p

import (
	"context"
	"testing"
)

func TestPing(t *testing.T) {
	ctx := context.Background()
	peers, err := NewTestNetwork(ctx, t, 3)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	if err := connectNodes(ctx, peers); err != nil {
		t.Errorf("error connecting peers: %s", err.Error())
	}

	for i, p1 := range peers {
		for _, p2 := range peers[i+1:] {
			lat, err := p1.Ping(p2.ID)
			if err != nil {
				t.Errorf("%s -> %s error: %s", p1.ID.Pretty(), p2.ID.Pretty(), err.Error())
				return
			}
			t.Logf("%s Ping: %s: %s", p1.ID, p2.ID, lat)
		}
	}
}
