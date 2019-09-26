package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	core "github.com/ipfs/go-ipfs/core"
	namesys "github.com/ipfs/go-ipfs/namesys"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	libp2p "github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	pstoremem "github.com/libp2p/go-libp2p-peerstore/pstoremem"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/qri-io/ioes"
	ipfs_filestore "github.com/qri-io/qfs/cafs/ipfs"
	"github.com/qri-io/qri/config"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo"
)

// QriNode encapsulates a qri peer-2-peer node
type QriNode struct {
	// ID is the node's identifier both locally & on the network
	// Identity has a relationship to privateKey (hash of PublicKey)
	ID peer.ID
	// private key for encrypted communication & verifying identity
	privateKey crypto.PrivKey

	cfg *config.P2P

	// base context for this node
	ctx context.Context

	// Online indicates weather this is node is connected to the p2p network
	Online bool
	// Host for p2p connections. can be provided by an ipfs node
	host host.Host
	// Discovery service, can be provided by an ipfs node
	Discovery discovery.Service

	// Repo is a repository of this node's qri data
	// note that repo's are built upon a cafs.Filestore, which
	// may contain a reference to a functioning IPFS node. In that case
	// QriNode should piggyback non-qri-specific p2p functionality on the
	// ipfs node provided by repo
	Repo repo.Repo

	// handlers maps this nodes registered handlers. This works in a way
	// similary to a router in traditional client/server models, but messages
	// are flying around all over the place instead of a
	// request/response pattern
	handlers map[MsgType]HandlerFunc

	// msgState keeps a "scratch pad" of message IDS & timeouts
	msgState *sync.Map
	// msgChan provides a channel of received messages for others to tune into
	msgChan chan Message
	// receivers is a list of anyone who wants to be notifed on new
	// message arrival
	receivers []chan Message

	// node keeps a set of IOStreams for "node local" io, often to the
	// command line, to give feedback to the user. These may be piped to
	// local http handlers/websockets/stdio, but these streams are meant for
	// local feedback as opposed to p2p connections
	LocalStreams ioes.IOStreams

	// networkNotifee satisfies the net.Notifee interface
	networkNotifee networkNotifee

	// TODO - waiting on next IPFS release
	// autoNAT service
	// autonat *autonat.AutoNATService
}

// Assert that conversions needed by the tests are valid.
var _ p2ptest.TestablePeerNode = (*QriNode)(nil)
var _ p2ptest.NodeMakerFunc = NewTestableQriNode

// NewTestableQriNode creates a new node, as a TestablePeerNode, usable by testing utilities.
func NewTestableQriNode(r repo.Repo, p2pconf *config.P2P) (p2ptest.TestablePeerNode, error) {
	return NewQriNode(r, p2pconf)
}

// NewQriNode creates a new node from a configuration. To get a fully connected
// node that's searching for peers call:
// n, _ := NewQriNode(r, cfg)
// n.GoOnline()
func NewQriNode(r repo.Repo, p2pconf *config.P2P) (node *QriNode, err error) {
	pid, err := p2pconf.DecodePeerID()
	if err != nil {
		return nil, fmt.Errorf("error decoding peer id: %s", err.Error())
	}

	node = &QriNode{
		ID:       pid,
		cfg:      p2pconf,
		Repo:     r,
		ctx:      context.Background(),
		msgState: &sync.Map{},
		msgChan:  make(chan Message),
		// Make sure we always have proper IOStreams, this can be set
		// later
		LocalStreams: ioes.NewDiscardIOStreams(),
	}
	node.handlers = MakeHandlers(node)

	// using this work around, rather than implimenting the Notifee
	// functions themselves, allows us to not pollute the QriNode
	// namespace with function names that we may want to use in the future
	node.networkNotifee = networkNotifee{node}

	return node, nil
}

// Host returns the node's Host
func (n *QriNode) Host() host.Host {
	return n.host
}

// setHost replaces the current host with the given host
// should only ever be called in GoOnline, when we already have a node
// but have not created a host yet
func (n *QriNode) setHost(h host.Host) {
	n.host = h
}

// GoOnline puts QriNode on the distributed web, ensuring there's an active peer-2-peer host
// participating in a peer-2-peer network, and kicks off requests to connect to known bootstrap
// peers that support the QriProtocol
func (n *QriNode) GoOnline() (err error) {
	if !n.cfg.Enabled {
		return fmt.Errorf("p2p connection is disabled")
	}

	if n.Online {
		return nil
	}
	// If the underlying content-addressed-filestore is an ipfs
	// node, it has built-in p2p, overlay the qri protocol
	// on the ipfs node's p2p connections.
	if ipfsfs, ok := n.Repo.Store().(*ipfs_filestore.Filestore); ok {
		if !ipfsfs.Online() {
			if err := ipfsfs.GoOnline(n.ctx); err != nil {
				return err
			}
		}

		ipfsnode := ipfsfs.Node()
		if ipfsnode.PeerHost != nil {
			n.host = ipfsnode.PeerHost
		}

		if ipfsnode.Discovery != nil {
			n.Discovery = ipfsnode.Discovery
		}
	} else if n.host == nil {
		ps := pstoremem.NewPeerstore()
		n.host, err = makeBasicHost(n.ctx, ps, n.cfg)
		if err != nil {
			return fmt.Errorf("error creating host: %s", err.Error())
		}
	}

	// add multistream handler for qri protocol to the host
	// setting a stream handler for the QriPrtocolID indicates to peers on
	// the distributed web that this node supports Qri. for more info on
	// multistreams  check github.com/multformats/go-multistream
	n.host.SetStreamHandler(QriProtocolID, n.QriStreamHandler)

	// TODO - wait for new IPFS release
	// if n.cfg.AutoNAT {
	// 	n.autonat, err = autonat.NewAutoNATService(n.ctx, n.host)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// add n.networkNotifee as a Notifee of this network
	n.host.Network().Notify(n.networkNotifee)

	p, err := n.Repo.Profile()
	if err != nil {
		log.Errorf("error getting repo profile: %s\n", err.Error())
		return err
	}
	p.PeerIDs = []peer.ID{n.host.ID()}

	// update profile with our p2p addresses
	if err := n.Repo.SetProfile(p); err != nil {
		return err
	}

	n.Online = true
	go n.echoMessages()

	return n.startOnlineServices()
}

// startOnlineServices bootstraps the node to qri & IPFS networks
// and begins NAT discovery
func (n *QriNode) startOnlineServices() error {
	if !n.Online {
		return nil
	}

	bsPeers := make(chan pstore.PeerInfo, len(n.cfg.BootstrapAddrs))

	go func() {
		// block until we have at least one successful bootstrap connection
		<-bsPeers

		if err := n.AnnounceConnected(n.Context()); err != nil {
			log.Infof("error announcing connected: %s", err.Error())
		}
	}()

	return n.StartDiscovery(bsPeers)
}

// ReceiveMessages adds a listener for newly received messages
func (n *QriNode) ReceiveMessages() chan Message {
	r := make(chan Message)
	n.receivers = append(n.receivers, r)
	return r
}

func (n *QriNode) echoMessages() {
	for {
		msg := <-n.msgChan
		for _, r := range n.receivers {
			r <- msg
		}
	}
}

// ipfsNode returns the internal IPFS node
func (n *QriNode) ipfsNode() (*core.IpfsNode, error) {
	if ipfsfs, ok := n.Repo.Store().(*ipfs_filestore.Filestore); ok {
		return ipfsfs.Node(), nil
	}
	return nil, fmt.Errorf("not using IPFS")
}

// IPFS exposes the core.IPFS node if one exists.
// This is currently required by things like remoteClient in other packages,
// which don't work properly with the CoreAPI implementation
func (n *QriNode) IPFS() (*core.IpfsNode, error) {
	return n.ipfsNode()
}

// GetIPFSNamesys returns a namesystem from IPFS
func (n *QriNode) GetIPFSNamesys() (namesys.NameSystem, error) {
	ipfsn, err := n.ipfsNode()
	if err != nil {
		return nil, err
	}
	return ipfsn.Namesys, nil
}

// note: both ipfs_filestore and ipfs_http have this method
type ipfsApier interface {
	IPFSCoreAPI() coreiface.CoreAPI
}

// IPFSCoreAPI returns a IPFS API interface instance
func (n *QriNode) IPFSCoreAPI() (coreiface.CoreAPI, error) {
	if apier, ok := n.Repo.Store().(ipfsApier); ok {
		return apier.IPFSCoreAPI(), nil
	}
	return nil, fmt.Errorf("not using IPFS")
}

// ListenAddresses gives the listening addresses of this node on the p2p network as
// a slice of strings
func (n *QriNode) ListenAddresses() ([]string, error) {
	maddrs := n.EncapsulatedAddresses()
	addrs := make([]string, len(maddrs))
	for i, maddr := range maddrs {
		addrs[i] = maddr.String()
	}
	return addrs, nil
}

// EncapsulatedAddresses returns a slice of full multaddrs for this node
func (n *QriNode) EncapsulatedAddresses() []ma.Multiaddr {
	// Build host multiaddress
	hostAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", n.host.ID().Pretty()))
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	res := make([]ma.Multiaddr, len(n.host.Addrs()))
	for i, a := range n.host.Addrs() {
		res[i] = a.Encapsulate(hostAddr)
	}

	return res
}

// Context returns this node's context
func (n *QriNode) Context() context.Context {
	if n.ctx == nil {
		n.ctx = context.Background()
	}
	return n.ctx
}

// TODO - finish. We need a proper termination & cleanup process
// func (n *QriNode) Close() error {
// 	if node, err := n.IPFSNode(); err == nil {
// 		return node.Close()
// 	}
// }

// makeBasicHost creates a LibP2P host from a NodeCfg
func makeBasicHost(ctx context.Context, ps pstore.Peerstore, p2pconf *config.P2P) (host.Host, error) {
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
		libp2p.EnableRelay(circuit.OptHop),
		// libp2p.Routing
	}

	// Let's talk about these options a bit. Most of the time, we will never
	// follow the code path that takes us to makeBasicHost. Usually, we will be
	// using the Host that comes with the ipfs node. But, let's say we want to not
	// use that ipfs host, or, we are in a testing situation, we will need to
	// create our own host. If we do not explicitly pass the host the options
	// for a ConnManager, it will use the NullConnManager, which doesn't actually
	// tag or manage any conns.
	// So instead, we pass in the libp2p basic ConnManager:
	opts = append(opts, libp2p.ConnectionManager(connmgr.NewConnManager(1000, 0, time.Millisecond)))

	return libp2p.New(ctx, opts...)
}

// SendMessage opens a stream & sends a message from p to one ore more peerIDs
func (n *QriNode) SendMessage(ctx context.Context, msg Message, replies chan Message, pids ...peer.ID) error {
	for _, peerID := range pids {
		if peerID == n.ID {
			// can't send messages to yourself, silly
			continue
		}

		s, err := n.host.NewStream(ctx, peerID, QriProtocolID)
		if err != nil {
			return fmt.Errorf("error opening stream: %s", err.Error())
		}
		defer s.Close()

		// now that we have a confirmed working connection
		// tag this peer as supporting the qri protocol in the connection manager
		n.host.ConnManager().TagPeer(peerID, qriSupportKey, qriSupportValue)

		ws := WrapStream(s)
		go n.handleStream(ws, replies)
		if err := ws.sendMessage(msg); err != nil {
			return err
		}
	}

	return nil
}

// QriStreamHandler is the handler we register with the multistream muxer
func (n *QriNode) QriStreamHandler(s net.Stream) {
	// defer s.Close()
	n.handleStream(WrapStream(s), nil)
}

// handleStream is a for loop which receives and handles messages
// When Message.HangUp is true, it exits. This will close the stream
// on one of the sides. The other side's receiveMessage() will error
// with EOF, thus also breaking out from the loop.
func (n *QriNode) handleStream(ws *WrappedStream, replies chan Message) {
	for {
		// Loop forever, receiving messages until the other end hangs up
		// or something goes wrong
		msg, err := ws.receiveMessage()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Debugf("error receiving message: %s", err.Error())
			break
		}

		if replies != nil {
			go func() { replies <- msg }()
		}
		go func() {
			n.msgChan <- msg
		}()

		handler, ok := n.handlers[msg.Type]
		if !ok {
			log.Infof("peer %s sent unrecognized message type '%s', hanging up", n.ID, msg.Type)
			break
		}

		if hangup := handler(ws, msg); hangup {
			break
		}
	}

	ws.stream.Close()
}

// Keys returns the KeyBook for the node.
func (n *QriNode) Keys() pstore.KeyBook {
	return n.host.Peerstore()
}

// Addrs returns the AddrBook for the node.
func (n *QriNode) Addrs() pstore.AddrBook {
	return n.host.Peerstore()
}

// SimplePeerInfo returns a PeerInfo with just the ID and Addresses.
func (n *QriNode) SimplePeerInfo() pstore.PeerInfo {
	return pstore.PeerInfo{
		ID:    n.host.ID(),
		Addrs: n.host.Addrs(),
	}
}

// MakeHandlers generates a map of MsgTypes to their corresponding handler functions
func MakeHandlers(n *QriNode) map[MsgType]HandlerFunc {
	return map[MsgType]HandlerFunc{
		MtPing:              n.handlePing,
		MtProfile:           n.handleProfile,
		MtDatasetInfo:       n.handleDataset,
		MtDatasets:          n.handleDatasetsList,
		MtConnected:         n.handleConnected,
		MtResolveDatasetRef: n.handleResolveDatasetRef,
		MtDatasetLog:        n.handleDatasetLog,
		MtQriPeers:          n.handleQriPeers,
		MtLogDiff:           n.handleLogDiff,
	}
}
