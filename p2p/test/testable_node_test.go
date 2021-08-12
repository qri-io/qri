package p2ptest

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"

	libp2p "github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	host "github.com/libp2p/go-libp2p-core/host"
	net "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
	pstoremem "github.com/libp2p/go-libp2p-peerstore/pstoremem"
)

// TestableNode satisfies the TestablePeerNode interface
// It is used for testing inside the p2ptest package
type TestableNode struct {
	host          host.Host
	cfg           *config.P2P
	Repo          repo.Repo
	localResolver dsref.Resolver
}

// TestQriProtocolID is the key used to set the stream handler to our
// testing protocol
const TestQriProtocolID = protocol.ID("/qri/_testing")
const TestQriSupportKey = "qri-support-test"
const TestQriConnManagerTag = "qri_test"
const TestQriConnManagerValue = 6

var ErrTestQriProtocolNotSupported = fmt.Errorf("test qri protocol not supported")

// Host returns the node's underlying host
func (n *TestableNode) Host() host.Host {
	return n.host
}

// SimpleAddrInfo returns the PeerInfo of the TestableNode
func (n *TestableNode) SimpleAddrInfo() peer.AddrInfo {
	return peer.AddrInfo{
		ID:    n.host.ID(),
		Addrs: n.host.Addrs(),
	}
}

func (n *TestableNode) TestStreamHandler(s net.Stream) {
	fmt.Println("stream handler called")
}

// GoOnline assumes the TestNode is not online, it will set
// the StreamHandler and updates our profile with the underlying peerIDs
func (n *TestableNode) GoOnline(ctx context.Context) error {
	// add multistream handler for qri protocol to the host
	// for more info on multistreams check github.com/multformats/go-multistream
	// Setting the StreamHandler with the TestQriProtocol will let other peers
	// know that we can speak the TestQriProtocol
	n.Host().SetStreamHandler(TestQriProtocolID, n.TestStreamHandler)

	p := n.Repo.Profiles().Owner(ctx)
	p.PeerIDs = []peer.ID{n.host.ID()}

	// update profile with our p2p addresses
	if err := n.Repo.Profiles().SetOwner(ctx, p); err != nil {
		return err
	}

	return nil
}

// NewTestableNode creates a testable node from a repo and a config.P2P
// it creates a basic host
func NewTestableNode(r repo.Repo, p2pconf *config.P2P, _ event.Publisher) (TestablePeerNode, error) {
	ctx := context.Background()
	ps := pstoremem.NewPeerstore()
	// this is essentially what is located in the p2p.makeBasicHost function
	pk, err := p2pconf.DecodePrivateKey()
	if err != nil {
		return nil, err
	}

	pid, err := p2pconf.DecodePeerID()
	if err != nil {
		return nil, err
	}
	ps.AddPrivKey(pid, pk)
	ps.AddPubKey(pid, pk.GetPublic())
	opts := []libp2p.Option{
		libp2p.ListenAddrs(p2pconf.Addrs...),
		libp2p.Identity(pk),
		libp2p.Peerstore(ps),
		libp2p.ConnectionManager(connmgr.NewConnManager(1000, 0, time.Millisecond)),
	}
	basicHost, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	localResolver := dsref.SequentialResolver(r.Dscache(), r)
	return &TestableNode{
		host:          basicHost,
		Repo:          r,
		cfg:           p2pconf,
		localResolver: localResolver,
	}, nil
}

var _ TestablePeerNode = (*TestableNode)(nil)
var _ NodeMakerFunc = NewTestableNode
