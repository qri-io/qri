package p2p

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo/profile"

	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	pstore "gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

// ConnectedQriProfiles lists all connected peers that support the qri protocol
func (n *QriNode) ConnectedQriProfiles() map[profile.ID]*config.ProfilePod {
	peers := map[profile.ID]*config.ProfilePod{}
	if n.Host == nil {
		return peers
	}
	for _, conn := range n.Host.Network().Conns() {
		if p, err := n.Repo.Profiles().PeerProfile(conn.RemotePeer()); err == nil {
			if pe, err := p.Encode(); err == nil {
				pe.Online = true
				peers[p.ID] = pe
			}
		}
	}
	return peers
}

// ConnectedQriPeerIDs returns a slice of peer.IDs this peer is currently connected to
func (n *QriNode) ConnectedQriPeerIDs() []peer.ID {
	peers := []peer.ID{}
	if n.Host == nil {
		return peers
	}
	conns := n.Host.Network().Conns()
	for _, c := range conns {
		id := c.RemotePeer()
		if _, err := n.Repo.Profiles().PeerProfile(id); err == nil {
			peers = append(peers, id)
		}
	}
	return peers
}

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

	if len(pid) == 0 {
		for _, conn := range n.Host.Network().Conns() {
			pid = append(pid, conn.RemotePeer())
			added++
			if added == max {
				break
			}
		}
	}

	return
}

// peerDifference returns a slice of peer IDs that are present in a but not b
func peerDifference(a, b []peer.ID) (diff []peer.ID) {
	m := make(map[peer.ID]bool)
	for _, bid := range b {
		m[bid] = true
	}

	for _, aid := range a {
		if _, ok := m[aid]; !ok {
			diff = append(diff, aid)
		}
	}
	return
}

// PeerInfo returns this peer's ID & Addresses as a peerstore.PeerInfo
func (n *QriNode) PeerInfo() pstore.PeerInfo {
	if !n.Online {
		return pstore.PeerInfo{}
	}

	return pstore.PeerInfo{
		ID:    n.Host.ID(),
		Addrs: n.Host.Addrs(),
	}
}

// AddQriPeer negotiates a connection with a peer to get their profile details
// and peer list.
func (n *QriNode) AddQriPeer(pinfo pstore.PeerInfo) error {
	// add this peer to our store
	n.Host.Peerstore().AddAddrs(pinfo.ID, pinfo.Addrs, pstore.TempAddrTTL)
	n.Host.ConnManager().TagPeer(pinfo.ID, qriConnManagerTag, qriConnManagerValue)

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
		if ti := n.Host.ConnManager().GetTagInfo(c.RemotePeer()); ti != nil {
			peers[i] = fmt.Sprintf("%s, %d, %v", c.RemotePeer().Pretty(), ti.Value, ti.Tags)
		}
	}

	return peers
}

// PeerConnectionParams defines parameters for the ConnectToPeer command
type PeerConnectionParams struct {
	Peername  string
	ProfileID profile.ID
	PeerID    peer.ID
	Multiaddr ma.Multiaddr
}

// ConnectToPeer takes a raw peer ID & tries to work out a route to that
// peer, explicitly connecting to them.
func (n *QriNode) ConnectToPeer(ctx context.Context, p PeerConnectionParams) (*profile.Profile, error) {
	log.Debugf("connect to peer: %v", p)
	pinfo, err := n.peerConnectionParamsToPeerInfo(p)
	if err != nil {
		return nil, err
	}

	if err := n.Host.Connect(ctx, pinfo); err != nil {
		return nil, err
	}

	if err := n.AddQriPeer(pinfo); err != nil {
		return nil, err
	}

	return n.Repo.Profiles().PeerProfile(pinfo.ID)
}

// DisconnectFromPeer explicitly closes a connection to a peer
func (n *QriNode) DisconnectFromPeer(ctx context.Context, p PeerConnectionParams) error {
	pinfo, err := n.peerConnectionParamsToPeerInfo(p)
	if err != nil {
		return err
	}

	return n.Host.Network().ClosePeer(pinfo.ID)
}

func (n *QriNode) peerConnectionParamsToPeerInfo(p PeerConnectionParams) (pi pstore.PeerInfo, err error) {
	if p.Multiaddr != nil {
		return toPeerInfos([]ma.Multiaddr{p.Multiaddr})[0], nil
	} else if len(p.PeerID) > 0 {
		return n.getPeerInfo(p.PeerID)
	}

	proID := p.ProfileID
	if len(proID) == 0 && p.Peername != "" {
		// TODO - there's lot's of possibile ambiguity around resolving peernames
		// this naive implementation for now just checks the profile store for a
		// matching peername
		proID, err = n.Repo.Profiles().PeernameID(p.Peername)
		if err != nil {
			return
		}
	}

	ids, err := n.Repo.Profiles().PeerIDs(proID)
	if err != nil {
		return
	}
	if len(ids) == 0 {
		return pstore.PeerInfo{}, fmt.Errorf("no network info for %s", proID)
	}

	// TODO - there's ambiguity here that we should address, for now
	// we'll just by default connect to the first peer
	return n.getPeerInfo(ids[0])
}

func (n *QriNode) getPeerInfo(pid peer.ID) (pstore.PeerInfo, error) {
	// first check for local peer info
	if pinfo := n.Host.Peerstore().PeerInfo(pid); len(pinfo.ID) > 0 {
		// _, err := n.RequestProfile(pinfo.ID)
		return pinfo, nil
	}

	// attempt to use ipfs routing table to discover peer
	ipfsnode, err := n.IPFSNode()
	if err != nil {
		log.Debug(err.Error())
		return pstore.PeerInfo{}, err
	}

	return ipfsnode.Routing.FindPeer(context.Background(), pid)
}
