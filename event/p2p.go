package event

var (
	// ETP2PGoneOnline occurs when a p2p node opens up for peer-2-peer connections
	// payload will be []multiaddr.Addr, the listening addresses of this peer
	ETP2PGoneOnline = Type("p2p:GoneOnline")
	// ETP2PGoneOffline occurs when a p2p node has finished disconnecting from
	// a peer-2-peer network
	// payload will be nil
	ETP2PGoneOffline = Type("p2p:GoneOffline")
	// ETP2PQriPeerConnected fires whenever a peer-2-peer connection that
	// supports the qri protocol is established
	// payload is a *profile.Profile
	// subscribers cannot block the publisher
	ETP2PQriPeerConnected = Type("p2p:QriPeerConnected")
	// ETP2PQriPeerDisconnected fires whenever a qri peer-2-peer connection
	// is closed
	// payload is a *profile.Profile
	// a nil payload means we never successfully obtained the peer's profile
	// information
	// subscribers cannot block the publisher
	ETP2PQriPeerDisconnected = Type("p2p:QriPeerDisconnected")
	// ETP2PPeerConnected occurs after any peer has connected to this node
	// payload will be a libp2p.peerInfo
	ETP2PPeerConnected = Type("p2p:PeerConnected")
	// ETP2PPeerDisconnected occurs after any peer has connected to this node
	// payload will be a libp2p.peerInfo
	ETP2PPeerDisconnected = Type("p2p:PeerDisconnected")
	// ETP2PMessageReceived fires whenever the p2p protocol receives a message
	// from a Qri peer
	// payload will be a p2p.Message
	ETP2PMessageReceived = Type("p2p:MessageReceived")
)
