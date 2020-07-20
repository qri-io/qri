package p2p

import (
	"context"
	"sync"
	"testing"

	// peerstore "github.com/libp2p/go-libp2p-core/peerstore"
	peer "github.com/libp2p/go-libp2p-core/peer"
	p2ptest "github.com/qri-io/qri/p2p/test"
)

func TestRequestProfileConnectNodes(t *testing.T) {
	t.Skip("TODO (ramfox): p2p tests are too flakey at the moment")
	ctx := context.Background()
	f := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, f, 5)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	if err := p2ptest.ConnectNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	peers := asQriNodes(testPeers)

	t.Logf("testing profile message with %d peers", len(peers))
	var wg sync.WaitGroup
	for i, p1 := range peers {
		for _, p2 := range peers[i+1:] {
			wg.Add(1)
			go func(p1, p2 *QriNode) {
				defer wg.Done()

				_, err := p1.RequestProfile(ctx, p2.ID)
				if err != nil {
					t.Errorf("%s -> %s error: %s", p1.ID.Pretty(), p2.ID.Pretty(), err.Error())
				}

				pro, err := p1.Repo.Profiles().PeerProfile(p2.ID)
				if err != nil {
					t.Errorf("error getting profile from profile store: %s", err.Error())
					return
				}

				if pro == nil {
					t.Error("profile shouldn't be nil")
					return
				}
				if len(pro.PeerIDs) == 0 {
					t.Error("profile should have peer IDs")
					return
				}

				addrsP1 := p1.host.Peerstore().Addrs(p2.ID)
				if len(addrsP1) == 0 {
					t.Errorf("%s (request node) should have addrs of %s (response node)", p1.ID, p2.ID)
				}
				addrsP2 := p2.host.Peerstore().Addrs(p1.ID)
				if len(addrsP2) == 0 {
					t.Errorf("%s (request node) should have addrs of %s (response node)", p2.ID, p1.ID)
				}

				pid := pro.PeerIDs[0]
				if err != nil {
					t.Error(err.Error())
					return
				}

				if pid != p2.ID {
					p2pro, _ := p2.Repo.Profile()
					t.Logf("p2 profile ID: %s peerID: %s, host peerID: %s", peer.ID(p2pro.ID), p2.ID, p2.host.ID())
					t.Errorf("%s request profile peerID mismatch. expected: %s, got: %s", p1.ID, p2.ID, pid)
				}

				pro1, err := p2.Repo.Profiles().PeerProfile(p1.ID)
				if err != nil {
					t.Errorf("error getting request profile from respond profile store: %s", err.Error())
					return
				}

				if pro1 == nil {
					t.Error("profile shouldn't be nil")
					return
				}
				if len(pro1.PeerIDs) == 0 {
					t.Error("profile should have peer IDs")
					return
				}

			}(p1, p2)
		}
	}

	wg.Wait()
}
