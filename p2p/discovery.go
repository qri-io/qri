package p2p

import (
	"context"
	"fmt"
	"time"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	discovery "gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/discovery"
)

const qriSupportKey = "qri-support"

// StartDiscovery initiates peer discovery, allocating a discovery
// services if one doesn't exist, then registering to be notified on peer discovery
func (n *QriNode) StartDiscovery() error {
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
	go n.Bootstrap(n.BootstrapAddrs)

	return nil
}

// HandlePeerFound
// TODO - TEST THIS. I suspect there's a bug in this implentation, or discovery
// notifications aren't working so well these days...
func (n *QriNode) HandlePeerFound(pinfo pstore.PeerInfo) {
	// first check to see if we've seen this peer before
	if _, err := n.Host.Peerstore().Get(pinfo.ID, qriSupportKey); err == nil {
		return
	} else if support, err := n.SupportsQriProtocol(pinfo.ID); err == nil {
		if err := n.Host.Peerstore().Put(pinfo.ID, qriSupportKey, support); err != nil {
			fmt.Println("error setting qri support flag", err.Error())
			return
		}

		if support {
			if err := n.AddQriPeer(pinfo); err != nil {
				fmt.Println(err.Error())
			}
		}
	} else if err != nil {
		fmt.Println("error checking for qri support:", err.Error())
	}
}

// SupportsQriProtocol checks to see if this peer supports the qri
// streaming protocol, returns
func (n *QriNode) SupportsQriProtocol(peer peer.ID) (bool, error) {
	protos, err := n.Host.Peerstore().GetProtocols(peer)
	if err == nil {
		for _, p := range protos {
			if p == string(QriProtocolId) {
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
