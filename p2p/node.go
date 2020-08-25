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
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	libp2pevent "github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	net "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	pstoremem "github.com/libp2p/go-libp2p-peerstore/pstoremem"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs/qipfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
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

	// QriProfileService allows your node to examine a peer & protect the
	// connections to that peer if it "speaks" the qri protocol
	// it also manages requests for that peer's qri profile and handles
	// requests for this node's profile
	qis *QriProfileService

	// localResolver allows the node to resolve local dataset references
	localResolver dsref.Resolver

	// handlers maps this nodes registered handlers. This works in a way
	// similary to a router in traditional client/server models, but messages
	// are flying around all over the place instead of a
	// request/response pattern
	handlers map[MsgType]HandlerFunc

	// msgState keeps a "scratch pad" of message IDS & timeouts
	msgState *sync.Map
	// receivers is a list of anyone who wants to be notifed on new
	// message arrival
	receivers []chan Message
	// receiversMu is the lock for the receivers list
	receiversMu sync.Mutex

	// pub is the event publisher on which to publish p2p events
	pub     event.Publisher
	notifee *net.NotifyBundle

	// node keeps a set of IOStreams for "node local" io, often to the
	// command line, to give feedback to the user. These may be piped to
	// local http handlers/websockets/stdio, but these streams are meant for
	// local feedback as opposed to p2p connections
	LocalStreams ioes.IOStreams

	// shutdown is the function used to cancel the context of the qri node
	shutdown context.CancelFunc
}

// Assert that conversions needed by the tests are valid.
var _ p2ptest.TestablePeerNode = (*QriNode)(nil)
var _ p2ptest.NodeMakerFunc = NewTestableQriNode

// NewTestableQriNode creates a new node, as a TestablePeerNode, usable by testing utilities.
func NewTestableQriNode(r repo.Repo, p2pconf *config.P2P, pub event.Publisher) (p2ptest.TestablePeerNode, error) {
	localResolver := dsref.SequentialResolver(r.Dscache(), r)
	return NewQriNode(r, p2pconf, pub, localResolver)
}

// NewQriNode creates a new node from a configuration. To get a fully connected
// node that's searching for peers call:
// n, _ := NewQriNode(r, cfg)
// n.GoOnline()
func NewQriNode(r repo.Repo, p2pconf *config.P2P, pub event.Publisher, localResolver dsref.Resolver) (node *QriNode, err error) {
	pid, err := p2pconf.DecodePeerID()
	if err != nil {
		return nil, fmt.Errorf("error decoding peer id: %s", err.Error())
	}

	node = &QriNode{
		ID:            pid,
		cfg:           p2pconf,
		Repo:          r,
		msgState:      &sync.Map{},
		pub:           pub,
		receiversMu:   sync.Mutex{},
		localResolver: localResolver,
		// Make sure we always have proper IOStreams, this can be set later
		LocalStreams: ioes.NewDiscardIOStreams(),
	}

	// TODO (ramfox): remove `MakeHandlers` when we phase out older p2p functions
	node.handlers = MakeHandlers(node)
	node.notifee = &net.NotifyBundle{
		ConnectedF:    node.connected,
		DisconnectedF: node.disconnected,
	}

	node.qis = NewQriProfileService(node.Repo, node.pub)
	return node, nil
}

// Host returns the node's Host
func (n *QriNode) Host() host.Host {
	return n.host
}

// GoOnline puts QriNode on the distributed web, ensuring there's an active peer-2-peer host
// participating in a peer-2-peer network, and kicks off requests to connect to known bootstrap
// peers that support the QriProtocol
func (n *QriNode) GoOnline(c context.Context) (err error) {
	ctx, cancel := context.WithCancel(c)
	n.shutdown = cancel

	log.Debugf("going online")
	if !n.cfg.Enabled {
		cancel()
		return fmt.Errorf("p2p connection is disabled")
	}
	if n.Online {
		cancel()
		return nil
	}

	// If the underlying content-addressed-filestore is an ipfs
	// node, it has built-in p2p, overlay the qri protocol
	// on the ipfs node's p2p connections.
	if ipfsfs, ok := n.Repo.Store().(*qipfs.Filestore); ok {
		log.Debugf("using IPFS p2p Host")
		if !ipfsfs.Online() {
			if err := ipfsfs.GoOnline(ctx); err != nil {
				cancel()
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
		log.Debugf("creating p2p Host")
		ps := pstoremem.NewPeerstore()
		n.host, err = makeBasicHost(ctx, ps, n.cfg)
		if err != nil {
			cancel()
			return fmt.Errorf("error creating host: %s", err.Error())
		}

		// we need to BYO discovery service when working without IPFS
		if err := n.setupDiscovery(ctx); err != nil {
			// we don't want to fail completely if discovery services like mdns aren't
			// supported. Otherwise routers with mdns turned off would break p2p entirely
			log.Errorf("couldn't start discovery: %s", err)
		}
	}

	n.qis.Start(n.host)

	// add multistream handler for qri protocol to the host
	// setting a stream handler for the QriPrtocolID indicates to peers on
	// the distributed web that this node supports Qri. for more info on
	// multistreams  check github.com/multformats/go-multistream
	// note: even though we are phasing out the old qri protocol, we
	// should still handle it for now, since this is what old nodes will
	// be relying on. We will drop support for the old qri protocol
	// in the release of v0.10.0
	n.host.SetStreamHandler(depQriProtocolID, n.QriStreamHandler)

	// add ref resolution capabilities:
	n.host.SetStreamHandler(ResolveRefProtocolID, n.resolveRefHandler)

	// register ourselves as a notifee on connected
	n.host.Network().Notify(n.notifee)
	if err := n.libp2pSubscribe(ctx); err != nil {
		cancel()
		return err
	}

	p, err := n.Repo.Profile()
	if err != nil {
		log.Errorf("error getting repo profile: %s\n", err.Error())
		cancel()
		return err
	}
	p.PeerIDs = []peer.ID{n.host.ID()}

	// update profile with our p2p addresses
	if err := n.Repo.SetProfile(p); err != nil {
		cancel()
		return err
	}

	n.Online = true
	n.pub.Publish(ctx, event.ETP2PGoneOnline, n.EncapsulatedAddresses())

	return n.startOnlineServices(ctx)
}

// startOnlineServices bootstraps the node to qri & IPFS networks
// and begins NAT discovery
func (n *QriNode) startOnlineServices(ctx context.Context) error {
	if !n.Online {
		return nil
	}
	log.Debugf("starting online services")

	// Boostrap off of default addresses
	go n.Bootstrap(n.cfg.QriBootstrapAddrs)
	// Bootstrap to IPFS network if this node is using an IPFS fs
	go n.BootstrapIPFS()
	return nil
}

// GoOffline takes the peer offline and shuts it down
func (n *QriNode) GoOffline() error {
	if n != nil && n.Online {
		err := n.Host().Close()
		// clean up the "GoOnline" context
		n.shutdown()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		n.pub.Publish(ctx, event.ETP2PGoneOffline, nil)
		n.Online = false
		return err
	}
	return nil
}

// ReceiveMessages adds a listener for newly received messages
func (n *QriNode) ReceiveMessages() chan Message {
	n.receiversMu.Lock()
	defer n.receiversMu.Unlock()
	r := make(chan Message)
	n.receivers = append(n.receivers, r)
	return r
}

func (n *QriNode) writeToReceivers(msg Message) {
	n.receiversMu.Lock()
	defer n.receiversMu.Unlock()
	for _, r := range n.receivers {
		r <- msg
	}
}

// IPFS exposes the core.IPFS node if one exists.
// This is currently required by things like remoteClient in other packages,
// which don't work properly with the CoreAPI implementation
func (n *QriNode) IPFS() (*core.IpfsNode, error) {
	if ipfsfs, ok := n.Repo.Store().(*qipfs.Filestore); ok {
		return ipfsfs.Node(), nil
	}
	return nil, fmt.Errorf("not using IPFS")
}

// GetIPFSNamesys returns a namesystem from IPFS
func (n *QriNode) GetIPFSNamesys() (namesys.NameSystem, error) {
	ipfsn, err := n.IPFS()
	if err != nil {
		return nil, err
	}
	return ipfsn.Namesys, nil
}

// note: both qipfs and ipfs_http have this method
type ipfsApier interface {
	IPFSCoreAPI() coreiface.CoreAPI
}

// IPFSCoreAPI returns a IPFS API interface instance
func (n *QriNode) IPFSCoreAPI() (coreiface.CoreAPI, error) {
	if n == nil {
		return nil, ErrNoQriNode
	}
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
	hostAddr, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", n.host.ID().Pretty()))
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

// makeBasicHost creates a LibP2P host from a NodeCfg
func makeBasicHost(ctx context.Context, ps peerstore.Peerstore, p2pconf *config.P2P) (host.Host, error) {
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
// TODO (ramfox): this relies on the soon to be removed `depQriProtocolID`
// one option: send message can be refactored to be protocol agnostic, so it can
// be used to send messages regardless of protocol.
func (n *QriNode) SendMessage(ctx context.Context, msg Message, replies chan Message, pids ...peer.ID) error {
	for _, peerID := range pids {
		if peerID == n.ID {
			// can't send messages to yourself, silly
			continue
		}

		s, err := n.host.NewStream(ctx, peerID, depQriProtocolID)
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

// connected is called when a connection opened via the network notifee bundle
func (n *QriNode) connected(_ net.Network, conn net.Conn) {
	log.Debugf("connected to peer: %s", conn.RemotePeer())
	pi := n.Host().Peerstore().PeerInfo(conn.RemotePeer())
	n.pub.Publish(context.Background(), event.ETP2PPeerConnected, pi)
}

func (n *QriNode) disconnected(_ net.Network, conn net.Conn) {
	pi := n.Host().Peerstore().PeerInfo(conn.RemotePeer())
	n.pub.Publish(context.Background(), event.ETP2PPeerDisconnected, pi)

	n.qis.HandleQriPeerDisconnect(pi.ID)
}

// QriStreamHandler is the handler we register with the multistream muxer
func (n *QriNode) QriStreamHandler(s net.Stream) {
	// defer s.Close()
	n.handleStream(WrapStream(s), nil)
}

func (n *QriNode) libp2pSubscribe(ctx context.Context) error {
	host := n.host
	sub, err := host.EventBus().Subscribe([]interface{}{
		new(libp2pevent.EvtPeerIdentificationCompleted),
		new(libp2pevent.EvtPeerIdentificationFailed),
	},
	// libp2peventbus.BufSize(1024),
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to identify notifications: %w", err)
	}
	go func() {
		defer sub.Close()
		for e := range sub.Out() {
			switch e := e.(type) {
			case libp2pevent.EvtPeerIdentificationCompleted:
				log.Debugf("libp2p identified peer: %s", e.Peer)
				n.qis.QriProfileRequest(ctx, e.Peer)
			case libp2pevent.EvtPeerIdentificationFailed:
				log.Debugf("libp2p failed to identify peer %s: %s", e.Peer, e.Reason)
			}
		}
	}()
	return nil
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

		handler, ok := n.handlers[msg.Type]
		if !ok {
			log.Infof("peer %s sent unrecognized message type '%s', hanging up", n.ID, msg.Type)
			break
		}

		n.pub.Publish(context.Background(), event.ETP2PMessageReceived, msg)
		go n.writeToReceivers(msg)

		if hangup := handler(ws, msg); hangup {
			break
		}
	}

	// ws.stream.Close()
}

// Keys returns the KeyBook for the node.
func (n *QriNode) Keys() peerstore.KeyBook {
	return n.host.Peerstore()
}

// Addrs returns the AddrBook for the node.
func (n *QriNode) Addrs() peerstore.AddrBook {
	return n.host.Peerstore()
}

// SimpleAddrInfo returns a PeerInfo with just the ID and Addresses.
func (n *QriNode) SimpleAddrInfo() peer.AddrInfo {
	return peer.AddrInfo{
		ID:    n.host.ID(),
		Addrs: n.host.Addrs(),
	}
}

// MakeHandlers generates a map of MsgTypes to their corresponding handler functions
func MakeHandlers(n *QriNode) map[MsgType]HandlerFunc {
	return map[MsgType]HandlerFunc{
		MtPing:     n.handlePing,
		MtProfile:  n.handleProfile,
		MtDatasets: n.handleDatasetsList,
		MtQriPeers: n.handleQriPeers,
	}
}
