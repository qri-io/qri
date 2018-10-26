package p2p

import (
	"context"
	"fmt"
	"time"

	discovery "gx/ipfs/QmY51bqSM5XgxQZqsBrQcRkKTnCb8EKpJpR9K6Qax7Njco/go-libp2p/p2p/discovery"
	pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
)

// StartDiscovery initiates peer discovery, allocating a discovery
// services if one doesn't exist, then registering to be notified on peer discovery
func (n *QriNode) StartDiscovery(bootstrapPeers chan pstore.PeerInfo) error {
	if n.Discovery == nil {
		service, err := discovery.NewMdnsService(context.Background(), n.host, time.Second*5, QriServiceTag)
		if err != nil {
			return err
		}
		n.Discovery = service
	}

	// Registering will call n.HandlePeerFound when peers are discovered
	n.Discovery.RegisterNotifee(n)

	// Check our existing peerstore for any potential friends
	go n.DiscoverPeerstoreQriPeers(n.host.Peerstore())
	// Boostrap off of default addresses
	go n.Bootstrap(n.cfg.BootstrapAddrs, bootstrapPeers)
	// Bootstrap to IPFS network if this node is using an IPFS fs
	go n.BootstrapIPFS()

	return nil
}

// HandlePeerFound deals with the discovery of a peer that may or may not support
// the qri protocol
func (n *QriNode) HandlePeerFound(pinfo pstore.PeerInfo) {
	log.Debugf("found peer %s", pinfo.ID)
	err := n.UpgradeToQriConnection(pinfo)
	if err != nil && err != ErrQriProtocolNotSupported {
		log.Error(err)
	}
}

// DiscoverPeerstoreQriPeers handles the case where a store has seen peers that
// support the qri protocol, but we haven't added them to our own peers list
func (n *QriNode) DiscoverPeerstoreQriPeers(store pstore.Peerstore) {
	for _, pid := range store.Peers() {
		if _, err := n.host.Peerstore().Get(pid, qriSupportKey); err == pstore.ErrNotFound {
			if supports, err := n.supportsQriProtocol(pid); err == nil && supports {
				// TODO - slow this down plz
				if err := n.UpgradeToQriConnection(store.PeerInfo(pid)); err != nil {
					fmt.Println(err.Error())
				}
			}
		}
	}
}
