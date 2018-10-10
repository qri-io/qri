package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo/profile"

	ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
)

// MtQriPeers is a request to get a list of known qri peers
var MtQriPeers = MsgType("qri_peers")

// QriPeer is a minimial struct that combines a profileID & network addresses
type QriPeer struct {
	ProfileID    string
	PeerID       string
	NetworkAddrs []string
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
		// TODO -
		ID, err := peer.IDB58Decode(strings.TrimPrefix(p.PeerID, "/ipfs/"))
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
func (n *QriNode) RequestQriPeers(id peer.ID) ([]QriPeer, error) {
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
	err = n.SendMessage(req, replies, id)
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
