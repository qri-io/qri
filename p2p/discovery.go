package p2p

import (
	"context"
	"fmt"
	"time"

	discovery "gx/ipfs/QmY51bqSM5XgxQZqsBrQcRkKTnCb8EKpJpR9K6Qax7Njco/go-libp2p/p2p/discovery"
	pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
)

// qriSupportKey is the key we store the flag for qri support under in Peerstores
const qriSupportKey = "qri-support"

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
	log.Debugf("In %s node, handling peer found %s", n.ID, pinfo.ID)
	log.Debug(n.host.Peerstore().PeerInfo(pinfo.ID))
	log.Debug(n.host.Network().Conns())
	// first check to see if we've seen this peer before
	if _, err := n.host.Peerstore().Get(pinfo.ID, qriSupportKey); err == nil {
		return
	}
	n.UpgradeToQriConnection(pinfo)
}

var errNoProtos = fmt.Errorf("no protocols available")

// SupportsQriProtocol checks to see if this peer supports the qri
// streaming protocol, returns
func (n *QriNode) SupportsQriProtocol(peer peer.ID) (bool, error) {
	protos, err := n.host.Peerstore().GetProtocols(peer)

	// if the list of protocols for this peer is empty, there's a good chance
	// we've not yet connected to them. Bailing on an empty slice of protos
	// has the effect of demanding we connect at least once before checking for
	// qri protocol support
	if len(protos) == 0 {
		return false, errNoProtos
	}

	if err == nil {
		for _, p := range protos {
			if p == string(QriProtocolID) {
				return true, nil
			}
		}
	}
	return false, err
}

// DiscoverPeerstoreQriPeers handles the case where a store has seen peers that
// support the qri protocol, but we haven't added them to our own peers list
func (n *QriNode) DiscoverPeerstoreQriPeers(store pstore.Peerstore) {
	for _, pid := range store.Peers() {
		if _, err := n.host.Peerstore().Get(pid, qriSupportKey); err == pstore.ErrNotFound {
			if supports, err := n.SupportsQriProtocol(pid); err == nil && supports {
				// TODO - slow this down plz
				if err := n.UpgradeToQriConnection(store.PeerInfo(pid)); err != nil {
					fmt.Println(err.Error())
				}
			}
		}
	}
}
