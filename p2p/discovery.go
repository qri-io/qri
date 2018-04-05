package p2p

import (
	"context"
	"fmt"
	"time"

	discovery "gx/ipfs/QmNh1kGFFdsPu79KNSaL4NUKUPb4Eiz4KHdMtFY6664RDp/go-libp2p/p2p/discovery"
	pstore "gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

// qriSupportKey is the key we store the flag for qri support under in Peerstores
const qriSupportKey = "qri-support"

// StartDiscovery initiates peer discovery, allocating a discovery
// services if one doesn't exist, then registering to be notified on peer discovery
func (n *QriNode) StartDiscovery(bootstrapPeers chan pstore.PeerInfo) error {
	if n.Discovery == nil {
		service, err := discovery.NewMdnsService(context.Background(), n.Host, time.Second*5, QriServiceTag)
		if err != nil {
			return err
		}
		n.Discovery = service
	}

	// Registering will call n.HandlePeerFound when peers are discovered
	n.Discovery.RegisterNotifee(n)

	// Check our existing peerstore for any potential friends
	go n.DiscoverPeerstoreQriPeers(n.Host.Peerstore())
	// Boostrap off of default addresses
	go n.Bootstrap(n.BootstrapAddrs, bootstrapPeers)
	// Bootstrap to IPFS network if this node is using an IPFS fs
	go n.BootstrapIPFS()

	return nil
}

// HandlePeerFound deals with the discovery of a peer that may or may not support
// the qri protocol
func (n *QriNode) HandlePeerFound(pinfo pstore.PeerInfo) {
	// first check to see if we've seen this peer before
	if _, err := n.Host.Peerstore().Get(pinfo.ID, qriSupportKey); err == nil {
		return
	} else if support, err := n.SupportsQriProtocol(pinfo.ID); err == nil {
		if err := n.Host.Peerstore().Put(pinfo.ID, qriSupportKey, support); err != nil {
			log.Errorf("error setting qri support flag", err.Error())
			return
		}

		if support {
			if err := n.AddQriPeer(pinfo); err != nil {
				log.Errorf("error adding qri peer: %s", err.Error())
			} else {
				log.Infof("discovered qri peer: %s", pinfo.ID)
			}
		}
	} else if err != nil && err != errNoProtos {
		log.Errorf("error checking for qri support:", err.Error())
	}
}

var errNoProtos = fmt.Errorf("no protocols available for check")

// SupportsQriProtocol checks to see if this peer supports the qri
// streaming protocol, returns
func (n *QriNode) SupportsQriProtocol(peer peer.ID) (bool, error) {
	protos, err := n.Host.Peerstore().GetProtocols(peer)

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
		if _, err := n.Host.Peerstore().Get(pid, qriSupportKey); err == pstore.ErrNotFound {
			if supports, err := n.SupportsQriProtocol(pid); err == nil && supports {
				// TODO - slow this down plz
				if err := n.AddQriPeer(store.PeerInfo(pid)); err != nil {
					fmt.Println(err.Error())
				}
			}
		}
	}
}
