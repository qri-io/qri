package p2p

import (
	"context"
	// "fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datatogether/api/apiutil"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry"
	inet "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
	// pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	// pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	// peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
)

// Convert from test nodes to non-test nodes.
func asQriNodes(testPeers []p2ptest.TestablePeerNode) []*QriNode {
	// Convert from test nodes to non-test nodes.
	peers := make([]*QriNode, len(testPeers))
	for i, node := range testPeers {
		peers[i] = node.(*QriNode)
	}
	return peers
}

func TestConnectedQriProfiles(t *testing.T) {
	ctx := context.Background()
	f := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, f)
	if err != nil {
		t.Error(err.Error())
		return
	}

	nodes := asQriNodes(testPeers)

	for i, a := range nodes {
		for _, b := range nodes[i+1:] {
			bpi := b.SimplePeerInfo()
			a.Host.Connect(ctx, bpi)
			a.HandlePeerFound(bpi)
		}
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

// MockReputationServer sets up a server that returns a given reputation each
// time the server is pinged. Remember to call `defer mockRegServer.Close()`
// any time you create a server
func MockReputationServer(rep int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiutil.WriteResponse(w, registry.Reputation{
			ProfileID: "MockReputation",
			Rep:       rep,
		})
	}))
}

func TestCheckReputation(t *testing.T) {
	ctx := context.Background()
	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestDirNetwork(ctx, factory)
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	nodes := asQriNodes(testPeers)

	single := nodes[0]
	group := nodes[1:]

	for _, node := range group {
		pinfo := node.SimplePeerInfo()
		if err := single.Host.Connect(single.Context(), pinfo); err != nil {
			t.Error(err.Error())
			return
		}
		if err := single.Host.Peerstore().Put(pinfo.ID, qriSupportKey, true); err != nil {
			t.Errorf("error setting qri support flag: %s", err.Error())
			return
		}
		if err := single.Host.Peerstore().Put(pinfo.ID, qriReputationKey, qriBaselineReputation); err != nil {
			t.Errorf("error setting qri reputation in peerstore: %s", err.Error())
		}
	}

	cases := []struct {
		reputation       int
		expectedTagValue int
		expectConnection bool
	}{
		{0, qriBaselineReputation + 0, true},
		{-1, qriBaselineReputation + -1, true},
		{-100, qriBaselineReputation + -100, false},
		{10, qriBaselineReputation + 10, true},
	}
	for i, c := range cases {
		mockRegServer := MockReputationServer(c.reputation)
		single.setRegistryLocation(mockRegServer.URL)
		for j, peer := range group {
			profile, err := peer.Repo.Profile()
			if err != nil {
				t.Errorf("case %d %s: %s", i, profile.ID, err)
				resetTagAndConn(t, single, peer, profile)
				continue
			}
			err = single.checkReputation(profile)
			if err != nil {
				t.Errorf("case %d %s: %s", i, profile.ID, err)
				resetTagAndConn(t, single, peer, profile)
				continue
			}
			checkTagAndConn(t, single, i, j, profile, c.expectedTagValue, c.expectConnection)
			resetTagAndConn(t, single, peer, profile)
		}
		mockRegServer.Close()
	}
}

func checkTagAndConn(t *testing.T, node *QriNode, caseNum, profileNum int, profile *profile.Profile, expectTagValue int, expectConnected bool) {
	// get & check connections:
	conns := []inet.Conn{}
	// get list of all conns
	for i, pid := range profile.PeerIDs {
		conns = append(conns, node.Host.Network().ConnsToPeer(pid)...)
		// get & check tag value
		gotTagValue := node.Host.ConnManager().GetTagInfo(pid).Tags[qriConnManagerTag]
		if expectTagValue != gotTagValue {
			t.Errorf("case %d profile %d tag %d, tag value mismatch. expected: %d, got: %d", caseNum, profileNum, i, expectTagValue, gotTagValue)
		}
	}
	// if connections are expected and there are no connections, error
	if expectConnected {
		if len(conns) == 0 {
			t.Errorf("case %d profile %d, connection mismatch, expected connections, got 0 connections", caseNum, profileNum)
		}
		return
	}
	// no connections are expected, if there are connections error
	if len(conns) != 0 {
		t.Errorf("case %d profile %d, connection mismatched, expected no connections, got %d connections", caseNum, profileNum, len(conns))
	}
}

func resetTagAndConn(t *testing.T, node, peer *QriNode, profile *profile.Profile) {
	// add address back to peerstore
	pinfo := peer.SimplePeerInfo()
	node.Host.Connect(node.Context(), pinfo)
	// node.Host.ConnManager().UntagPeer(pinfo.ID, qriConnManagerTag)
	if err := node.Host.Peerstore().Put(pinfo.ID, qriReputationKey, qriBaselineReputation); err != nil {
		t.Errorf("error setting qri reputation in peerstore: %s", err.Error())
	}
}
