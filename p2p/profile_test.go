package p2p

import (
	"context"
	"sync"
	"testing"
)

func TestRequestProfile(t *testing.T) {
	ctx := context.Background()
	peers, err := NewTestNetwork(ctx, t, 5)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	if err := connectNodes(ctx, peers); err != nil {
		t.Errorf("error connecting peers: %s", err.Error())
	}

	t.Logf("testing profile message with %d peers", len(peers))
	var wg sync.WaitGroup
	for i, p1 := range peers {
		for _, p2 := range peers[i+1:] {
			wg.Add(1)
			go func(p1, p2 *QriNode) {
				defer wg.Done()

				pro, err := p1.RequestProfile(p2.ID)
				if err != nil {
					t.Errorf("%s -> %s error: %s", p1.ID.Pretty(), p2.ID.Pretty(), err.Error())
				}
				if pro == nil {
					t.Error("profile shouldn't be nil")
					return
				}
				pid, err := pro.IPFSPeerID()
				if err != nil {
					t.Error(err.Error())
					return
				}
				if pid != p2.ID {
					t.Errorf("profile id mismatch. expected: %s, got: %s", p2.ID.Pretty(), pro.ID)
				}
				t.Log(pro)
			}(p1, p2)
		}
	}

	wg.Wait()
}
