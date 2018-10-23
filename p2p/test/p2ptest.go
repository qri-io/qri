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

	ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	host "gx/ipfs/Qmb8T6YBBsjYsVGfrihQLfCJveczZnneSBqBKkYEBWDjge/go-libp2p-host"
)

// TestablePeerNode is used by tests only. Implemented by QriNode
type TestablePeerNode interface {
	// add Host here?
	Host() host.Host
	SimplePeerInfo() pstore.PeerInfo
	UpgradeToQriConnection(pstore.PeerInfo) error
	GoOnline() error
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
	if err := node.GoOnline(); err != nil {
		return nil, fmt.Errorf("errror connecting: %s", err.Error())
	}
	node.Host().Peerstore().AddPubKey(info.PeerID, info.PubKey)
	node.Host().Peerstore().AddPrivKey(info.PeerID, info.PrivKey)
	return node, err
}

// ConnectNodes creates a basic connection between the nodes. This connection
// mirrors the connection that would normally occur between two p2p nodes.
// The Host.Connect function adds the addresses into the peerstore and dials the
// remote peer.
// Take a look at https://github.com/libp2p/go-libp2p-host/blob/623ffaa4ef2b8dad77933159d0848a393a91c41e/host.go#L36
// for more info
// Connect should always:
// - add a connections to the peer
// - add the addrs of the peer to the peerstore
// - add a tag for each peer in the connmanager
func ConnectNodes(ctx context.Context, nodes []TestablePeerNode) error {
	var wg sync.WaitGroup
	connect := func(n TestablePeerNode, pinfo pstore.PeerInfo) error {
		if err := n.Host().Connect(ctx, pinfo); err != nil {
			return fmt.Errorf("error connecting nodes: %s", err)
		}
		wg.Done()
		return nil
	}

	for i, s1 := range nodes {
		for _, s2 := range nodes[i+1:] {
			wg.Add(1)
			if err := connect(s1, s2.SimplePeerInfo()); err != nil {
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
		if err := a.UpgradeToQriConnection(bpi); err != nil {
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
