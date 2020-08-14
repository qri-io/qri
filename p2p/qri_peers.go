package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo/profile"
)

// MtQriPeers is a request to get a list of known qri peers
var MtQriPeers = MsgType("qri_peers")

// QriPeer is a minimial struct that combines a profileID & network addresses
type QriPeer struct {
	ProfileID    string
	PeerID       string
	NetworkAddrs []string
}

// upgradeToQriConnection attempts to open a Qri protocol connection to a peer
// it records whether the peer supports Qri in the host Peerstore,
// returns ErrQriProtocolNotSupported if the connection cannot be upgraded,
// and sets a priority in the host Connection Manager if the connection is upgraded
func (n *QriNode) upgradeToQriConnection(pid peer.ID) error {
	// bail early if we have seen this peer before
	log.Debugf("%s, attempting to upgrading %s to qri connection", n.ID, pid)
	if _support, err := n.host.Peerstore().Get(pid, qriSupportKey); err == nil {
		support, ok := _support.(bool)
		if !ok {
			return fmt.Errorf("support flag stored incorrectly in the peerstore")
		}
		if support {
			return nil
		}
	}

	// check if this connection supports the qri protocol
	support, err := n.supportsDepQriProtocol(pid)
	if err != nil {
		log.Debugf("error checking for qri support: %s", err)
		return err
	}
	// mark whether or not this connection supports the qri protocol:
	if err := n.host.Peerstore().Put(pid, qriSupportKey, support); err != nil {
		log.Debugf("error setting qri support flag: %s", err)
		return err
	}
	// if it does support the qri protocol
	// - tag the connection as a qri connection in the ConnManager
	// - request profile
	// - request profiles
	if !support {
		log.Debugf("%s could not upgrade %s to Qri connection: %s", n.ID, pid, ErrQriProtocolNotSupported)
		return ErrQriProtocolNotSupported
	}
	log.Debugf("%s upgraded %s to Qri connection", n.ID, pid)
	// tag the connection as more important in the conn manager:
	n.host.ConnManager().TagPeer(pid, qriSupportKey, qriSupportValue)

	ctx, done := context.WithTimeout(context.Background(), time.Second*20)
	defer done()

	pro, err := n.RequestProfile(ctx, pid)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	n.pub.Publish(ctx, event.ETP2PQriPeerConnected, pro)

	go func() {
		ctx, done := context.WithTimeout(context.Background(), time.Second*20)
		defer done()
		ps, err := n.RequestQriPeers(ctx, pid)
		if err != nil {
			log.Debug("error fetching qri peers: %s", err)
		}
		n.RequestNewPeers(ctx, ps)
	}()

	return nil
}

// supportsDepQriProtocol checks to see if this peer supports the old qri
// streaming protocol
func (n *QriNode) supportsDepQriProtocol(peer peer.ID) (bool, error) {
	protos, err := n.host.Peerstore().GetProtocols(peer)
	if err != nil {
		return false, err
	}

	for _, p := range protos {
		if p == string(depQriProtocolID) {
			return true, nil
		}
	}
	return false, nil
}

func toQriPeers(psm map[profile.ID]*config.ProfilePod) (peers []QriPeer) {
	for id, pp := range psm {
		p := QriPeer{
			ProfileID:    id.String(),
			NetworkAddrs: pp.NetworkAddrs,
		}
		if len(pp.PeerIDs) >= 1 {
			p.PeerID = pp.PeerIDs[0]
		}
		peers = append(peers, p)
	}
	return
}

// RequestNewPeers intersects a provided list of peer info with this node's existing
// connections and attempts to connect to any peers it doesn't have connections to
func (n *QriNode) RequestNewPeers(ctx context.Context, peers []QriPeer) {
	var newPeers []QriPeer
	connected := n.ConnectedQriProfiles()
	for _, p := range peers {
		proID, err := profile.NewB58ID(p.ProfileID)
		if err != nil {
			continue
		}

		if connected[proID] != nil {
			continue
		}
		newPeers = append(newPeers, p)
	}

	for _, p := range newPeers {
		ID, err := peer.IDB58Decode(strings.TrimPrefix(strings.TrimPrefix(p.PeerID, "/ipfs/"), "/p2p/"))
		if err != nil {
			continue
		}

		ms, err := ParseMultiaddrs(p.NetworkAddrs)
		if err != nil {
			continue
		}

		var m ma.Multiaddr
		if len(ms) > 0 {
			m = ms[0]
		}

		go n.ConnectToPeer(ctx, PeerConnectionParams{
			PeerID:    ID,
			Multiaddr: m,
		})
	}
}

// RequestQriPeers asks a designated peer for a list of qri peers
func (n *QriNode) RequestQriPeers(ctx context.Context, id peer.ID) ([]QriPeer, error) {
	log.Debugf("%s RequestQriPeers: %s", n.ID, id)

	if id == n.ID {
		// requesting self isn't a network operation
		return toQriPeers(n.ConnectedQriProfiles()), nil
	}

	if !n.Online {
		return nil, fmt.Errorf("not connected to p2p network")
	}

	req, err := NewJSONBodyMessage(n.ID, MtQriPeers, nil)
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	req = req.WithHeaders("phase", "request")

	replies := make(chan Message)
	err = n.SendMessage(ctx, req, replies, id)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("send dataset info message error: %s", err.Error())
	}

	res := <-replies
	peers := []QriPeer{}
	err = json.Unmarshal(res.Body, &peers)
	return peers, err
}

func (n *QriNode) handleQriPeers(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true
	switch msg.Header("phase") {
	case "request":
		ps := toQriPeers(n.ConnectedQriProfiles())
		reply, err := msg.UpdateJSON(ps)
		if err != nil {
			log.Debug(err.Error())
			return
		}

		reply = reply.WithHeaders("phase", "response")
		if err := ws.sendMessage(reply); err != nil {
			log.Debug(err.Error())
			return
		}
	}
	return
}
