package p2p

import (
	"context"
	"fmt"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// AddQriPeer negotiates a connection with a peer to get their profile details
// and peer list.
func (n *QriNode) AddQriPeer(pinfo pstore.PeerInfo) error {
	// add this peer to our store
	n.QriPeers.AddAddrs(pinfo.ID, pinfo.Addrs, pstore.TempAddrTTL)

	// if profile, _ := n.Repo.Peers().GetPeer(pinfo.ID); profile != nil {
	// 	// we've already seen this peer
	// 	return nil
	// }

	if err := n.RequestProfileInfo(pinfo); err != nil {
		return err
	}

	// some time later ask for a list of their peers, you know, "for a friend"
	go func() {
		// time.Sleep(time.Second * 2)
		n.RequestPeersList(pinfo.ID)
	}()

	return nil
}

// RequestPeername attempts to find profile info for a given peername
func (n *QriNode) RequestPeername(peername string) error {
	return nil
}

// RequestProfileInfo get's qri profile information from a PeerInfo
func (n *QriNode) RequestProfileInfo(pinfo pstore.PeerInfo) error {
	// Get this repo's profile information
	profile, err := n.Repo.Profile()
	if err != nil {
		fmt.Println("error getting node profile info:", err)
		return err
	}

	addrs, err := n.IPFSListenAddresses()
	if err != nil {
		return err
	}
	profile.Addresses = addrs

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

// RequestPeersList asks a peer for a list of peers they've seen
func (n *QriNode) RequestPeersList(id peer.ID) {
	res, err := n.SendMessage(id, &Message{
		Type: MtPeers,
		Payload: &PeersReqParams{
			Offset: 0,
			Limit:  10,
		},
	})

	if err != nil {
		fmt.Println("send peers message error:", err.Error())
		return
	}

	if res.Phase == MpResponse {
		if err := n.handlePeersResponse(res); err != nil {
			fmt.Println("peers response error", err.Error())
			return
		}
	}
}

// ConnectToPeer takes a raw peer ID & tries to work out a route to that
// peer, explicitly connecting to them.
func (n *QriNode) ConnectToPeer(pid peer.ID) error {
	// first check for local peer info
	if pinfo := n.Host.Peerstore().PeerInfo(pid); pinfo.ID.String() != "" {
		return n.RequestProfileInfo(pinfo)
	}

	// attempt to use ipfs routing table to discover peer
	ipfsnode, err := n.IPFSNode()
	if err != nil {
		return err
	}

	pinfo, err := ipfsnode.Routing.FindPeer(context.Background(), pid)
	if err != nil {
		return err
	}

	return n.RequestProfileInfo(pinfo)
}

// ConnectedPeers lists all IPFS connected peers
func (n *QriNode) ConnectedPeers() []string {
	conns := n.Host.Network().Conns()
	peers := make([]string, len(conns))
	for i, c := range conns {
		peers[i] = c.RemotePeer().Pretty()
	}

	return peers
}
