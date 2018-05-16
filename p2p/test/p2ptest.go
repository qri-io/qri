package p2ptest

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"

	testutil "gx/ipfs/QmVvkK7s5imCiq3JVbL3pGfnhcCnf3LrFJPF4GE2sAoGZf/go-testutil"
	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	pstore "gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	net "gx/ipfs/QmXfkENeeBvh3zYA51MaSdGUdBjhQ99cP5WQe8zgr6wchG/go-libp2p-net"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	"testing"
)

// TestablePeerNode is used by tests only. Implemented by QriNode
type TestablePeerNode interface {
	Keys() pstore.KeyBook
	Addrs() pstore.AddrBook
	HostNetwork() net.Network
	SimplePeerInfo() pstore.PeerInfo
	AddPeer(pstore.PeerInfo) error
}

// NodeMakerFunc is a function that constructs a Node from a Repo and options.
type NodeMakerFunc func(repo.Repo, ...func(*config.P2P)) (TestablePeerNode, error)

// NewTestNetwork constructs nodes to test p2p functionality.
func NewTestNetwork(ctx context.Context, t *testing.T, num int, maker NodeMakerFunc) ([]TestablePeerNode, error) {
	nodes := make([]TestablePeerNode, num)

	for i := 0; i < num; i++ {
		rid, err := testutil.RandPeerID()
		if err != nil {
			return nil, fmt.Errorf("error creating peer ID: %s", err.Error())
		}

		r, err := test.NewTestRepoFromProfileID(profile.ID(rid), i, i)
		if err != nil {
			return nil, fmt.Errorf("error creating test repo: %s", err.Error())
		}

		node, err := NewTestNode(r, t, maker)
		if err != nil {
			return nil, err
		}

		nodes[i] = node
	}
	return nodes, nil
}

// NewTestNode constructs a node for testing.
func NewTestNode(r repo.Repo, t *testing.T, maker NodeMakerFunc) (TestablePeerNode, error) {
	localnp := testutil.RandPeerNetParamsOrFatal(t)
	data, err := localnp.PrivKey.Bytes()
	if err != nil {
		return nil, err
	}

	privKey := base64.StdEncoding.EncodeToString(data)

	node, err := maker(r, func(c *config.P2P) {
		c.PeerID = localnp.ID.Pretty()
		c.PrivKey = privKey
		c.Addrs = []ma.Multiaddr{
			localnp.Addr,
		}
		c.QriBootstrapAddrs = []string{}
	})
	if err != nil {
		return nil, fmt.Errorf("error creating test node: %s", err.Error())
	}
	node.Keys().AddPubKey(localnp.ID, localnp.PubKey)
	node.Keys().AddPrivKey(localnp.ID, localnp.PrivKey)

	return node, err
}

// NewTestDirNetwork constructs nodes from the testdata directory, for p2p testing
func NewTestDirNetwork(ctx context.Context, t *testing.T, maker NodeMakerFunc) ([]TestablePeerNode, error) {
	dirs, err := ioutil.ReadDir("testdata")
	if err != nil {
		return nil, err
	}

	nodes := []TestablePeerNode{}
	for _, dir := range dirs {
		if dir.IsDir() {
			repo, _, err := test.NewMemRepoFromDir(filepath.Join("testdata", dir.Name()))
			if err != nil {
				return nil, err
			}

			node, err := NewTestNode(repo, t, maker)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

// ConnectNodes connects the nodes in the network so they may communicate
func ConnectNodes(ctx context.Context, nodes []TestablePeerNode) error {
	var wg sync.WaitGroup
	connect := func(n TestablePeerNode, dst peer.ID, addr ma.Multiaddr) error {
		n.Addrs().AddAddr(dst, addr, pstore.PermanentAddrTTL)
		if _, err := n.HostNetwork().DialPeer(ctx, dst); err != nil {
			return err
		}
		wg.Done()
		return nil
	}

	for i, s1 := range nodes {
		for _, s2 := range nodes[i+1:] {
			wg.Add(1)
			if err := connect(s1, s2.HostNetwork().LocalPeer(), s2.HostNetwork().ListenAddresses()[0]); err != nil {
				return err
			}
		}
	}
	wg.Wait()

	return nil
}

// ConnectQriPeers connects the nodes as Qri peers using PeerInfo
func ConnectQriPeers(ctx context.Context, nodes []TestablePeerNode) error {
	var wg sync.WaitGroup
	connect := func(a, b TestablePeerNode) error {
		bpi := b.SimplePeerInfo()
		if err := a.AddPeer(bpi); err != nil {
			return err
		}
		wg.Done()
		return nil
	}

	for i, s1 := range nodes {
		for _, s2 := range nodes[i+1:] {
			wg.Add(1)
			if err := connect(s1, s2); err != nil {
				return err
			}
		}
	}
	wg.Wait()

	return nil
}
