// Package p2ptest defines utilities for qri peer-2-peer testing
package p2ptest

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/qri-io/dag"
	"github.com/qri-io/qri/config"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
	"github.com/qri-io/qri/repo/test"
)

// TestablePeerNode is used by tests only. Implemented by QriNode
type TestablePeerNode interface {
	Host() host.Host
	SimpleAddrInfo() peer.AddrInfo
	UpgradeToQriConnection(peer.AddrInfo) error
	GoOnline(ctx context.Context) error
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
	p2pconf := config.DefaultP2P()
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
// each of these peers has no datasets and no peers are connected
func NewTestNetwork(ctx context.Context, f *TestNodeFactory, num int) ([]TestablePeerNode, error) {
	nodes := make([]TestablePeerNode, num)
	for i := 0; i < num; i++ {
		info := f.NextInfo()
		r, err := test.NewTestRepoFromProfileID(profile.IDFromPeerID(info.PeerID), i, i)
		if err != nil {
			return nil, fmt.Errorf("error creating test repo: %s", err.Error())
		}
		node, err := NewAvailableTestNode(ctx, r, f)
		if err != nil {
			return nil, err
		}
		nodes[i] = node
	}
	return nodes, nil
}

// NewTestDirNetwork constructs nodes from the testdata directory, for p2p testing
// Peers are pulled from the "testdata" directory, and come pre-populated with datasets
// no peers are connected.
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

			node, err := NewAvailableTestNode(ctx, repo, f)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

// NewAvailableTestNode constructs a test node that is hooked up and ready to Connect
func NewAvailableTestNode(ctx context.Context, r repo.Repo, f *TestNodeFactory) (TestablePeerNode, error) {
	info := f.NextInfo()
	addr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	p2pconf := config.DefaultP2P()
	p2pconf.Addrs = []ma.Multiaddr{addr}
	p2pconf.QriBootstrapAddrs = []string{}
	node, err := f.NewWithConf(r, p2pconf)
	if err != nil {
		return nil, fmt.Errorf("error creating test node: %s", err.Error())
	}
	if err := node.GoOnline(ctx); err != nil {
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
// Take a look at https://github.com/libp2p/go-libp2p-core/host/blob/623ffaa4ef2b8dad77933159d0848a393a91c41e/host.go#L36
// for more info
// Connect should always:
// - add a connections to the peer
// - add the addrs of the peer to the peerstore
// - add a tag for each peer in the connmanager
func ConnectNodes(ctx context.Context, nodes []TestablePeerNode) error {
	var wg sync.WaitGroup
	connect := func(n TestablePeerNode, pinfo peer.AddrInfo) error {
		if err := n.Host().Connect(ctx, pinfo); err != nil {
			return fmt.Errorf("error connecting nodes: %s", err)
		}
		wg.Done()
		return nil
	}

	for i, s1 := range nodes {
		for _, s2 := range nodes[i+1:] {
			wg.Add(1)
			if err := connect(s1, s2.SimpleAddrInfo()); err != nil {
				return err
			}
		}
	}
	wg.Wait()
	return nil
}

// ConnectQriNodes takes a slice of unconnected nodes and returns a slice
// of connected nodes that have upgraded qri connections:
// They support the qri protocol and have exchanged profile
func ConnectQriNodes(ctx context.Context, nodes []TestablePeerNode) error {
	var wgConnect sync.WaitGroup
	connect := func(n TestablePeerNode, pinfo peer.AddrInfo) error {
		if err := n.Host().Connect(ctx, pinfo); err != nil {
			return fmt.Errorf("error connecting nodes: %s", err)
		}
		wgConnect.Done()
		return nil
	}

	for i, s1 := range nodes {
		for _, s2 := range nodes[i+1:] {
			wgConnect.Add(1)
			if err := connect(s1, s2.SimpleAddrInfo()); err != nil {
				return err
			}
		}
	}
	wgConnect.Wait()
	time.Sleep(time.Millisecond * 300)
	// previously, we had UpgradeToQriConnection running in separate threads
	// much like we did with the basic connection
	// however, UpgradeToQriConnection asks for and sends profile information
	// from it's various peers. We were running into a race condition where
	// we would be writing to and requesting a profile at the same time.
	for _, s1 := range nodes {
		for _, s2 := range nodes {
			pinfo := s2.SimpleAddrInfo()
			if s1.SimpleAddrInfo().ID == pinfo.ID {
				continue
			}
			if err := s1.UpgradeToQriConnection(pinfo); err != nil {
				return fmt.Errorf("%s error upgrading connection to %s: %s", s1.SimpleAddrInfo().ID, pinfo.ID, err)
			}
		}
	}

	return nil
}

// GetSomeBlocks returns a list of num ids for blocks that are in the referenced dataset.
func GetSomeBlocks(capi coreiface.CoreAPI, ref reporef.DatasetRef, num int) []string {
	result := []string{}

	ctx := context.Background()

	ng := dag.NewNodeGetter(capi.Dag())

	id, err := cid.Parse(ref.Path)
	if err != nil {
		panic(err)
	}

	elem, err := ng.Get(ctx, id)
	if err != nil {
		panic(err)
	}

	for _, link := range elem.Links() {
		result = append(result, link.Cid.String())
		if len(result) >= num {
			break
		}
	}

	return result
}
