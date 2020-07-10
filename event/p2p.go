package event

var (
	// ETP2PGoneOnline occurs when a p2p node opens up for peer-2-peer connections
	// payload will be []multiaddr.Addr, the listening addresses of this peer
	ETP2PGoneOnline = Type("p2p:GoneOnline")
	// ETP2PGoneOffline occurs when a p2p node has finished disconnecting from
	// a peer-2-peer network
	// payload will be nil
	// NOTE: not currently firing
	ETP2PGoneOffline = Type("p2p:GoneOffline")
	// ETP2PQriPeerConnected occurs after a qri peer has connected to this node
	// payload will be a fully hydrated *profile.Profile from
	// "github.com/qri-io/qri/repo/profile"
	// NOTE: not currently firing
	ETP2PQriPeerConnected = Type("p2p:QriPeerConnected")
	// ETP2PQriPeerDisconnected occurs after a qri peer has connected to this node
	// payload will be a fully hydrated *profile.Profile from
	// "github.com/qri-io/qri/repo/profile"
	// NOTE: not currently firing
	ETP2PQriPeerDisconnected = Type("p2p:QriPeerDisconnected")
	// ETP2PPeerConnected occurs after any peer has connected to this node
	// payload will be a libp2p.peerInfo
	// NOTE: not currently firing
	ETP2PPeerConnected = Type("p2p:PeerConnected")
	// ETP2PPeerDisconnected occurs after any peer has connected to this node
	// payload will be a libp2p.peerInfo
	// NOTE: not currently firing
	ETP2PPeerDisconnected = Type("p2p:PeerDisconnected")
	// ETP2PMessageReceived fires whenever the p2p protocol receives a message
	// from a Qri peer
	// payload will be a p2p.Message
	// NOTE: not currently firing
	ETP2PMessageReceived = Type("p2p:MessageReceived")
)
