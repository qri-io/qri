package p2p

import (
	"context"
	"sync"
	"testing"

	"github.com/qri-io/qri/p2p/test"
	// "github.com/qri-io/qri/repo/profile"

	pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
)

func TestRequestProfileConnectNodes(t *testing.T) {
	ctx := context.Background()
	f := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, f, 5)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}
	// Convert from test nodes to non-test nodes.
	peers := make([]*QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*QriNode)
	}

	if err := p2ptest.ConnectNodes(ctx, testPeers); err != nil {
		t.Errorf("error connecting peers: %s", err.Error())
	}

	t.Logf("testing profile message with %d peers", len(peers))
	var wg sync.WaitGroup
	for i, p1 := range peers {
		for _, p2 := range peers[i+1:] {
			wg.Add(1)
			go func(p1, p2 *QriNode) {
				defer wg.Done()

				_, err := p1.RequestProfile(p2.ID)
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

				addrsP1 := p1.Host.Peerstore().Addrs(p2.ID)
				if len(addrsP1) == 0 {
					t.Errorf("%s (request node) should have addrs of %s (response node)", p1.ID, p2.ID)
				}
				addrsP2 := p2.Host.Peerstore().Addrs(p1.ID)
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
					t.Logf("p2 profile ID: %s peerID: %s, host peerID: %s", peer.ID(p2pro.ID), p2.ID, p2.Host.ID())
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

func TestRequestProfileOneWayConnection(t *testing.T) {
	ctx := context.Background()
	f := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, f, 5)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	// Convert from test nodes to QriNodes.
	peers := make([]*QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*QriNode)
	}

	p1 := peers[0]
	peers = peers[1:]

	for _, peer := range peers {
		p1.Addrs().AddAddr(peer.HostNetwork().LocalPeer(), peer.HostNetwork().ListenAddresses()[0], pstore.PermanentAddrTTL)
	}

	t.Logf("testing profile message with %d peers", len(peers))
	for _, p2 := range peers {
		t.Logf("Getting profile from peer %s", p2.ID)
		_, err := p1.RequestProfile(p2.ID)
		if err != nil {
			t.Errorf("%s -> %s error: %s", p1.ID.Pretty(), p2.ID.Pretty(), err.Error())
		}
		pro, err := p1.Repo.Profiles().PeerProfile(p2.ID)
		if err != nil {
			t.Errorf("error getting profile from profile store: %s", err.Error())
			continue
		}

		if pro == nil {
			t.Error("profile shouldn't be nil")
			continue
		}
		if len(pro.PeerIDs) == 0 {
			t.Error("profile should have peer IDs")
			continue
		}

		peerInfo2 := p1.Host.Peerstore().PeerInfo(p2.ID)
		if len(peerInfo2.Addrs) == 0 {
			t.Errorf("%s (request node) should have addrs of %s (response node)", p1.ID, p2.ID)
		}
		peerInfo1 := p2.Host.Peerstore().PeerInfo(p1.ID)
		if len(peerInfo1.Addrs) == 0 {
			t.Errorf("%s (response node) should have addrs of %s (request node)", p2.ID, p1.ID)
		}

		pid := pro.PeerIDs[0]

		if pid != p2.ID {
			p2pro, _ := p2.Repo.Profile()
			t.Logf("p2 profile ID: %s peerID: %s, host peerID: %s", peer.ID(p2pro.ID), p2.ID, p2.Host.ID())
			t.Errorf("%s request profile peerID mismatch. expected: %s, got: %s", p1.ID, p2.ID, pid)
		}

		pro1, err := p2.Repo.Profiles().List()
		if err != nil {
			t.Errorf("error getting request profile from response profile store: %s", err.Error())
			continue
		}

		if pro1 == nil {
			t.Error("profile shouldn't be nil")
			continue
		}
		if len(pro1) == 0 {
			t.Error("profile should have peer IDs")
			continue
		}
	}
}
