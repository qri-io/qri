package p2p

import (
	"context"
	"fmt"
	"time"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	discovery "gx/ipfs/QmRQ76P5dgvxTujhfPsCRAG83rC15jgb1G9bKLuomuC6dQ/go-libp2p/p2p/discovery"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

const qriSupportKey = "qri-support"

// StartDiscovery initiates peer discovery, allocating a discovery
// services if one doesn't exist, then registering to be notified on peer discovery
func (n *QriNode) StartDiscovery() error {
	if n.Discovery == nil {
		service, err := discovery.NewMdnsService(context.Background(), n.Host, time.Second*5)
		if err != nil {
			return err
		}
		n.Discovery = service
	}

	// Registering will call n.HandlePeerFound when peers are discovered
	n.Discovery.RegisterNotifee(n)

	go n.DiscoverPeerstoreQriPeers(n.Host.Peerstore())

	return nil
}

// HandlePeerFound
// TODO - TEST THIS. suspect there's a bug in the implentation
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

func (n *QriNode) AddQriPeer(pinfo pstore.PeerInfo) error {
	// add this peer to our store
	n.QriPeers.AddAddrs(pinfo.ID, pinfo.Addrs, pstore.TempAddrTTL)

	if profile, _ := n.Repo.Peers().GetPeer(pinfo.ID); profile != nil {
		// we've already seen this peer
		return nil
	}

	// Get this repo's profile information
	profile, err := n.Repo.Profile()
	if err != nil {
		fmt.Println("error getting node profile info:", err)
		return err
	}

	res, err := n.SendMessage(pinfo.ID, &Message{
		Type:    MtPeerInfo,
		Payload: profile,
	})
	if err != nil {
		fmt.Println("send profile message error:", err.Error())
		return err
	}

	if res.Phase == MpResponse {
		if err := n.handleProfileResponse(pinfo, res); err != nil {
			fmt.Println("profile response error", err.Error())
			return err
		}
	}

	return nil
}

// DiscoverPeerstoreQriPeers handles the case where a store has seen peers that
// support the qri protocol, but we haven't added them to our own peers list
func (n *QriNode) DiscoverPeerstoreQriPeers(store pstore.Peerstore) {
	for _, pid := range store.Peers() {
		res, err := n.Host.Peerstore().Get(pid, qriSupportKey)
		fmt.Println(pid.Pretty(), res, err)

		if supports, err := n.SupportsQriProtocol(pid); err == nil && supports {
			// TODO - slow this down plz
			if err := n.AddQriPeer(store.PeerInfo(pid)); err != nil {
				fmt.Println(err.Error())
			}
		}
	}
}
