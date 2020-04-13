package p2p

import (
	"context"
	"time"

	peer "github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-core/peerstore"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery"
)

// startDiscovery initiates local peer discovery, allocating a discovery service
// if one doesn't exist, then registering to be notified on peer discovery
func (n *QriNode) startDiscovery() error {
	if n.Discovery == nil {
		service, err := discovery.NewMdnsService(context.Background(), n.host, time.Second*5, QriServiceTag)
		if err != nil {
			return err
		}
		n.Discovery = service
	}

	// Registering will call n.HandlePeerFound when peers are discovered
	n.Discovery.RegisterNotifee(n)
	return nil
}

// HandlePeerFound deals with the discovery of a peer that may or may not support
// the qri protocol
func (n *QriNode) HandlePeerFound(pinfo peer.AddrInfo) {
	log.Debugf("found peer %s", pinfo.ID)
	err := n.UpgradeToQriConnection(pinfo)
	if err != nil && err != ErrQriProtocolNotSupported {
		log.Error(err)
	}
}

// DiscoverPeerstoreQriPeers handles the case where a store has seen peers that
// support the qri protocol, but we haven't added them to our own peers list
func (n *QriNode) DiscoverPeerstoreQriPeers(store peerstore.Peerstore) {
	for _, pid := range store.Peers() {
		if _, err := n.host.Peerstore().Get(pid, qriSupportKey); err == peerstore.ErrNotFound {
			if supports, err := n.supportsQriProtocol(pid); err == nil && supports {
				// TODO - slow this down plz
				if err := n.UpgradeToQriConnection(store.PeerInfo(pid)); err != nil {
					log.Debug(err.Error())
				}
			}
		}
	}
}
