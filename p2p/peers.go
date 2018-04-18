package p2p

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/repo/profile"

	pstore "gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

// ClosestConnectedPeers checks if a peer is connected, and if so adds it to the top
// of a slice cap(max) of peers to try to connect to
// TODO - In the future we'll use a few tricks to improve on just iterating the list
// at a bare minimum we should grab a randomized set of peers
func (n *QriNode) ClosestConnectedPeers(id profile.ID, max int) (pid []peer.ID) {
	added := 0
	if !n.Online {
		return []peer.ID{}
	}

	if ids, err := n.Repo.Profiles().PeerIDs(id); err == nil {
		for _, id := range ids {
			if len(n.Host.Network().ConnsToPeer(id)) > 0 {
				added++
				pid = append(pid, id)
			}
		}
	}

	for _, conn := range n.Host.Network().Conns() {
		pid = append(pid, conn.RemotePeer())
		added++
		if added == max {
			break
		}
	}

	return
}

// AddQriPeer negotiates a connection with a peer to get their profile details
// and peer list.
func (n *QriNode) AddQriPeer(pinfo pstore.PeerInfo) error {
	// add this peer to our store
	n.QriPeers.AddAddrs(pinfo.ID, pinfo.Addrs, pstore.TempAddrTTL)

	if _, err := n.RequestProfile(pinfo.ID); err != nil {
		log.Debug(err.Error())
		return err
	}

	return nil
}

// Peers returns a list of currently connected peer IDs
func (n *QriNode) Peers() []peer.ID {
	if n.Host == nil {
		return []peer.ID{}
	}
	conns := n.Host.Network().Conns()
	seen := make(map[peer.ID]struct{})
	peers := make([]peer.ID, 0, len(conns))

	for _, c := range conns {
		p := c.LocalPeer()
		if _, found := seen[p]; found {
			continue
		}

		seen[p] = struct{}{}
		peers = append(peers, p)
	}

	return peers
}

// ConnectedPeers lists all IPFS connected peers
func (n *QriNode) ConnectedPeers() []string {
	if n.Host == nil {
		return []string{}
	}
	conns := n.Host.Network().Conns()
	peers := make([]string, len(conns))
	for i, c := range conns {
		peers[i] = c.RemotePeer().Pretty()
	}

	return peers
}

// ConnectToPeer takes a raw peer ID & tries to work out a route to that
// peer, explicitly connecting to them.
func (n *QriNode) ConnectToPeer(pid peer.ID) error {
	// first check for local peer info
	if pinfo := n.Host.Peerstore().PeerInfo(pid); pinfo.ID.String() != "" {
		_, err := n.RequestProfile(pinfo.ID)
		return err
	}

	// attempt to use ipfs routing table to discover peer
	ipfsnode, err := n.IPFSNode()
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	pinfo, err := ipfsnode.Routing.FindPeer(context.Background(), pid)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	s, err := n.Host.NewStream(n.Context(), pinfo.ID, QriProtocolID)
	if err != nil {
		return fmt.Errorf("error opening stream: %s", err.Error())
	}
	s.Close()

	return nil
}
