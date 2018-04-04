package p2p

import (
	"context"
	"sync"
	"testing"

	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
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

				pid := pro.PeerIDs()[0]
				if err != nil {
					t.Error(err.Error())
					return
				}

				if pid != p2.ID {
					p2pro, _ := p2.Repo.Profile()
					t.Logf("p2 profile ID: %s peerID: %s, host peerID: %s", peer.ID(p2pro.ID), p2.ID, p2.Host.ID())
					t.Errorf("%s request profile peerID mismatch. expected: %s, got: %s", p1.ID, p2.ID, pid)
				}

			}(p1, p2)
		}
	}

	wg.Wait()
}
