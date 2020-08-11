package p2p

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo/profile"

	peer "github.com/libp2p/go-libp2p-core/peer"
	swarm "github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
)

// ConnectedQriProfiles lists all connected peers that support the qri protocol
func (n *QriNode) ConnectedQriProfiles() map[profile.ID]*config.ProfilePod {
	peers := map[profile.ID]*config.ProfilePod{}
	if n.host == nil {
		return peers
	}
	for _, conn := range n.host.Network().Conns() {
		if p, err := n.Repo.Profiles().PeerProfile(conn.RemotePeer()); err == nil {
			if pe, err := p.Encode(); err == nil {
				pe.Online = true
				// Build host multiaddress,
				// TODO - this should be a convenience func
				hostAddr, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", conn.RemotePeer().Pretty()))
				if err != nil {
					log.Debug(err.Error())
					return nil
				}

				pe.NetworkAddrs = []string{
					conn.RemoteMultiaddr().Encapsulate(hostAddr).String(),
				}
				peers[p.ID] = pe
			}
		}
	}
	return peers
}

// ConnectedQriPeerIDs returns a slice of peer.IDs this peer is currently connected to
func (n *QriNode) ConnectedQriPeerIDs() []peer.ID {
	return n.profiles.ConnectedQriPeers()
}

// ClosestConnectedQriPeers checks if a peer is connected, and if so adds it to the top
// of a slice cap(max) of peers to try to connect to
// TODO - In the future we'll use a few tricks to improve on just iterating the list
// at a bare minimum we should grab a randomized set of peers
func (n *QriNode) ClosestConnectedQriPeers(profileID profile.ID, max int) (pid []peer.ID) {
	added := 0
	if !n.Online {
		return []peer.ID{}
	}

	if peerIDs, err := n.Repo.Profiles().PeerIDs(profileID); err == nil {
		for _, peerID := range peerIDs {
			if len(n.host.Network().ConnsToPeer(peerID)) > 0 {
				added++
				pid = append(pid, peerID)
			}
		}
	}

	if len(pid) == 0 {
		for _, conn := range n.host.Network().Conns() {
			peerID := conn.RemotePeer()
			protocols, err := n.host.Peerstore().SupportsProtocols(peerID, string(QriProtocolID))
			if err != nil {
				continue
			}
			if len(protocols) != 0 {
				pid = append(pid, peerID)
				added++
				if added >= max {
					break
				}
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
func (n *QriNode) PeerInfo(pid peer.ID) peer.AddrInfo {
	if !n.Online {
		return peer.AddrInfo{}
	}

	return n.host.Peerstore().PeerInfo(pid)
}

// Peers returns a list of currently connected peer IDs
func (n *QriNode) Peers() []peer.ID {
	if n.host == nil {
		return []peer.ID{}
	}
	conns := n.host.Network().Conns()
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
	if n.host == nil {
		return []string{}
	}
	conns := n.host.Network().Conns()
	peers := make([]string, len(conns))
	for i, c := range conns {
		peers[i] = c.RemotePeer().Pretty()
		if ti := n.host.ConnManager().GetTagInfo(c.RemotePeer()); ti != nil {
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

	if swarm, ok := n.host.Network().(*swarm.Swarm); ok {
		// clear backoff b/c we're explicitly dialing this peer
		swarm.Backoff().Clear(pinfo.ID)
	}

	if err := n.host.Connect(ctx, pinfo); err != nil {
		return nil, fmt.Errorf("host connect %s failure: %s", pinfo.ID.Pretty(), err)
	}

	// do an explicit connection upgrade
	// TODO (ramfox): when we refactor this, we should really
	// be handing back a channel that will notify us when
	// the profile exchange is complete, so we know it's safe
	// to request the profile
	if err := n.upgradeToQriConnection(pinfo.ID); err != nil {
		if err == ErrQriProtocolNotSupported {
			return nil, fmt.Errorf("upgrading p2p connection to a qri connection: %w", err)
		}
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

	conns := n.host.Network().ConnsToPeer(pinfo.ID)
	for _, conn := range conns {
		if err := conn.Close(); err != nil {
			return err
		}
	}

	return nil
}

// peerConnectionParamsToPeerInfo turns connection parameters into something p2p can dial
func (n *QriNode) peerConnectionParamsToPeerInfo(p PeerConnectionParams) (pi peer.AddrInfo, err error) {
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
		return peer.AddrInfo{}, fmt.Errorf("no network info for %s", proID)
	}

	// TODO - there's ambiguity here that we should address, for now
	// we'll just by default connect to the first peer
	return n.getPeerInfo(ids[0])
}

// getPeerInfo first looks for local peer info, then tries to fall back to using IPFS
// to do routing lookups
func (n *QriNode) getPeerInfo(pid peer.ID) (peer.AddrInfo, error) {
	// first check for local peer info
	if pinfo := n.host.Peerstore().PeerInfo(pid); len(pinfo.ID) > 0 {
		// _, err := n.RequestProfile(pinfo.ID)
		return pinfo, nil
	}

	// attempt to use ipfs routing table to discover peer
	ipfsnode, err := n.IPFS()
	if err != nil {
		log.Debug(err.Error())
		return peer.AddrInfo{}, err
	}

	return ipfsnode.Routing.FindPeer(context.Background(), pid)
}
