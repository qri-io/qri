package p2ptest

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/qri-io/qri/config"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"

	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	pstore "gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	net "gx/ipfs/QmXfkENeeBvh3zYA51MaSdGUdBjhQ99cP5WQe8zgr6wchG/go-libp2p-net"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
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
type NodeMakerFunc func(repo.Repo, *config.P2P) (TestablePeerNode, error)

// TestNodeFactory can be used to safetly construct nodes for tests
type TestNodeFactory struct {
	count int
	maker NodeMakerFunc
}

// NewTestNodeFactory returns a new TestNodeFactory
func NewTestNodeFactory(maker NodeMakerFunc) *TestNodeFactory {
	return &TestNodeFactory{count: 0, maker: maker}
}

// New creates a new Node for testing
func (f *TestNodeFactory) New(r repo.Repo) (TestablePeerNode, error) {
	info := cfgtest.GetTestPeerInfo(f.count)
	f.count++
	p2pconf := config.NewP2P()
	p2pconf.PeerID = info.EncodedPeerID
	p2pconf.PrivKey = info.EncodedPrivKey
	return f.maker(r, p2pconf)
}

// NewWithConf creates a new Node for testing using a configuration
func (f *TestNodeFactory) NewWithConf(r repo.Repo, p2pconf *config.P2P) (TestablePeerNode, error) {
	info := cfgtest.GetTestPeerInfo(f.count)
	f.count++
	p2pconf.PeerID = info.EncodedPeerID
	p2pconf.PrivKey = info.EncodedPrivKey
	return f.maker(r, p2pconf)
}

// NextInfo gets the PeerInfo for the next test Node to be constructed
func (f *TestNodeFactory) NextInfo() *cfgtest.PeerInfo {
	return cfgtest.GetTestPeerInfo(f.count)
}

// NewTestNetwork constructs nodes to test p2p functionality.
func NewTestNetwork(ctx context.Context, f *TestNodeFactory, num int) ([]TestablePeerNode, error) {
	nodes := make([]TestablePeerNode, num)
	for i := 0; i < num; i++ {
		info := f.NextInfo()
		r, err := test.NewTestRepoFromProfileID(profile.ID(info.PeerID), i, i)
		if err != nil {
			return nil, fmt.Errorf("error creating test repo: %s", err.Error())
		}
		node, err := NewAvailableTestNode(r, f)
		if err != nil {
			return nil, err
		}
		nodes[i] = node
	}
	return nodes, nil
}

// NewTestDirNetwork constructs nodes from the testdata directory, for p2p testing
func NewTestDirNetwork(ctx context.Context, f *TestNodeFactory) ([]TestablePeerNode, error) {
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

			node, err := NewAvailableTestNode(repo, f)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

// NewAvailableTestNode constructs a test node that is hooked up and ready to Connect
func NewAvailableTestNode(r repo.Repo, f *TestNodeFactory) (TestablePeerNode, error) {
	info := f.NextInfo()
	addr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	p2pconf := config.NewP2P()
	p2pconf.Addrs = []ma.Multiaddr{addr}
	p2pconf.QriBootstrapAddrs = []string{}
	node, err := f.NewWithConf(r, p2pconf)
	if err != nil {
		return nil, fmt.Errorf("error creating test node: %s", err.Error())
	}
	node.Keys().AddPubKey(info.PeerID, info.PubKey)
	node.Keys().AddPrivKey(info.PeerID, info.PrivKey)
	return node, err
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
