package p2p

import (
	"context"
	"fmt"
	"github.com/qri-io/qri/repo"

	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	// net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	ping "github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ma "github.com/multiformats/go-multiaddr"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

// QriNode encapsulates a qri distributed node
type QriNode struct {
	Identity   peer.ID        // the local node's identity
	PrivateKey crypto.PrivKey // the local node's private Key

	Host      host.Host // p2p Host
	Pings     *ping.PingService
	Discovery discovery.Service

	repo repo.Repo
}

// NewQriNode creates a new node, providing no arguments will use
// default configuration
func NewQriNode(options ...func(o *NodeCfg)) (*QriNode, error) {
	cfg := DefaultNodeCfg()
	for _, opt := range options {
		opt(cfg)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	host, err := makeBasicHost(cfg)
	if err != nil {
		return nil, err
	}

	node := &QriNode{
		Identity: cfg.PeerId,
		Host:     host,
	}

	host.SetStreamHandler(ProtocolId, node.MessageStreamHandler)

	if err = node.StartDiscovery(); err != nil {
		return nil, err
	}

	return node, nil
}

// func (n *QriNode) Ping() (time.Duration, error) {
//   dur, err := n.Pings.Ping(context.Background(), p)
// }

// TODO - Finish
// func (n *QriNode) startOnlineServices(cfg *NodeCfg) error {
// // if n
// // n.Pings = ping.NewPingService(n.Host)

// // setup local discovery
// }

// Repo gives this node's repository
func (n *QriNode) Repo() repo.Repo {
	return n.repo
}

// Encapsulated Addresses returns a slice of full multaddrs for this node
func (qn *QriNode) EncapsulatedAddresses() []ma.Multiaddr {
	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", qn.Host.ID().Pretty()))

	res := make([]ma.Multiaddr, len(qn.Host.Addrs()))
	for i, a := range qn.Host.Addrs() {
		res[i] = a.Encapsulate(hostAddr)
	}

	return res
}

// makeBasicHost creates a LibP2P host from a NodeCfg
func makeBasicHost(cfg *NodeCfg) (host.Host, error) {
	// Create a peerstore
	ps := pstore.NewPeerstore()

	// If using secio, we add the keys to the peerstore
	// for this peer ID.
	if cfg.Secure {
		ps.AddPrivKey(cfg.PeerId, cfg.PrivKey)
		ps.AddPubKey(cfg.PeerId, cfg.PubKey)
	}

	// Set up stream multiplexer
	tpt := msmux.NewBlankTransport()
	tpt.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)

	// Create swarm (implements libP2P Network)
	swrm, err := swarm.NewSwarmWithProtector(
		context.Background(),
		cfg.Addrs,
		cfg.PeerId,
		ps,
		nil,
		tpt,
		nil,
	)
	if err != nil {
		return nil, err
	}

	netw := (*swarm.Network)(swrm)
	basicHost := bhost.New(netw)
	return basicHost, nil
}
