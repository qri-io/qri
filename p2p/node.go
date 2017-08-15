package p2p

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"

	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	ma "github.com/multiformats/go-multiaddr"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

// QriNode encapsulates a qri distributed node
type QriNode struct {
	Identity peer.ID   // the local node's identity
	Host     host.Host // p2p Host
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

	host.SetStreamHandler(ProtocolId, node.StreamHandler)
	return node, nil
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

// StreamHandler handles connections to this node
func (qn *QriNode) StreamHandler(s net.Stream) {
	defer s.Close()

	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Printf("received: %s\n", str)
	_, err = s.Write([]byte("PONG"))
	if err != nil {
		log.Println(err)
		return
	}
}

// Send Bytes to a given multiaddr
func (qn *QriNode) SendByteMsg(multiaddr string, msg []byte) (res []byte, err error) {
	addr, err := ma.NewMultiaddr(multiaddr)
	if err != nil {
		err = fmt.Errorf("invalid multiaddr: %s", err.Error())
		return
	}

	pid, err := addr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		return
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		return
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	if err != nil {
		return
	}

	targetAddr := addr.Decapsulate(targetPeerAddr)

	// We have a peer ID and a targetAddr so we add it to the peerstore
	// so LibP2P knows how to contact it
	qn.Host.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)
	s, err := qn.Host.NewStream(context.Background(), peerid, ProtocolId)
	if err != nil {
		return
	}
	defer s.Close()

	if _, err = s.Write(msg); err != nil {
		return
	}

	return ioutil.ReadAll(s)
}
