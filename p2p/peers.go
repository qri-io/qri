package p2p

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo/profile"

	ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
	swarm "gx/ipfs/QmemVjhp1UuWPQqrWSvPcaqH3QJRMjMqNm4T2RULMkDDQe/go-libp2p-swarm"
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
				// Build host multiaddress,
				// TODO - this should be a convenience func
				hostAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", conn.RemotePeer().Pretty()))
				if err != nil {
					log.Debug(err.Error())
					return nil
				}

				pe.NetworkAddrs = []string{conn.RemoteMultiaddr().Encapsulate(hostAddr).String()}
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

// PeerInfo returns peer peer ID & network multiaddrs from the Host Peerstore
func (n *QriNode) PeerInfo(pid peer.ID) pstore.PeerInfo {
	if !n.Online {
		return pstore.PeerInfo{}
	}

	return n.Host.Peerstore().PeerInfo(pid)
}

// AddQriPeer negotiates a connection with a peer to get their profile details
// and peer list.
func (n *QriNode) AddQriPeer(pinfo pstore.PeerInfo) error {
	// add this peer to our store so libp2p has the provided addresses of
	// the peer in the next call
	n.Host.Peerstore().AddAddrs(pinfo.ID, pinfo.Addrs, pstore.TempAddrTTL)

	if _, err := n.RequestProfile(pinfo.ID); err != nil {
		log.Debug(err.Error())
		return err
	}

	go func() {
		ps, err := n.RequestQriPeers(pinfo.ID)
		if err != nil {
			log.Debugf("error fetching qri peers: %s", err)
			return
		}
		n.RequestNewPeers(n.ctx, ps)
	}()

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

	if swarm, ok := n.Host.Network().(*swarm.Swarm); ok {
		// clear backoff b/c we're explicitly dialing this peer
		swarm.Backoff().Clear(pinfo.ID)
	}

	if err := n.Host.Connect(ctx, pinfo); err != nil {
		return nil, fmt.Errorf("host connect %s failure: %s", pinfo.ID.Pretty(), err)
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

	conns := n.Host.Network().ConnsToPeer(pinfo.ID)
	for _, conn := range conns {
		if err := conn.Close(); err != nil {
			return err
		}
	}

	return nil
}

// peerConnectionParamsToPeerInfo turns connection parameters into something p2p can dial
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

// getPeerInfo first looks for local peer info, then tries to fall back to using IPFS
// to do routing lookups
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
