package p2p

import (
	"context"
	// "github.com/qri-io/qri/repo/profile"

	// pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// MtProfiles is a request to get profile information from connected peers
var MtProfiles = MsgType("profiles")

// PeersReqParams outlines params for requesting peers
type PeersReqParams struct {
	Limit  int
	Offset int
}

// InitiateProfilesRequest kicks off a network wide scan for profiles
func (n *QriNode) InitiateProfilesRequest(ctx context.Context, id peer.ID) error {
	// pro, err := n.Repo.Profile()
	// if err != nil {
	// 	log.Debug(err.Error())
	// 	return err
	// }

	// pids := []peer.ID{}
	// for _, conn := range n.Host.Network().Conns() {
	// 	pids = append(pids, conn.RemotePeer())
	// }

	// replies := make(chan Message)
	// defer close(replies)

	// NewMessage(n.ID, MtProfiles, payload)

	// if err := n.SendMessage(msg, replies, pids...); err != nil {
	// 	log.Debug(err.Error())
	// 	return err
	// }

	return nil
}

func (n *QriNode) handleProfiles(ws *WrappedStream, msg Message) (hangup bool) {
	// p := &PeersReqParams{}
	// if err := json.Unmarshal(data, p); err != nil {
	// 	log.Debug("unmarshal peers request error:", err.Error())
	// 	return
	// }

	// ws.stream.Conn().RemotePeer()

	// pro, err := n.Repo.Profile()
	// if err != nil {
	// 	log.Debug(err.Error())
	// 	return true
	// }

	// if err := ws.sendMessage(msg.Update(pro)); err != nil {
	// 	return true
	// }

	return

	// profiles, err := repo.QueryPeers(n.Repo.Peers(), query.Query{
	// 	Limit:  p.Limit,
	// 	Offset: p.Offset,
	// })

	// if err != nil {
	// 	log.Info("error getting peer profiles:", err.Error())
	// 	return nil
	// }

	// return &Message{
	// 	Type:    MtProfiles,
	// 	Phase:   MpResponse,
	// 	Payload: profiles,
	// }
}

// func (n *QriNode) handlePeersResponse(r *Message) error {
// 	data, err := json.Marshal(r.Payload)
// 	if err != nil {
// 		return err
// 	}
// 	peers := []*profile.Profile{}
// 	if err := json.Unmarshal(data, &peers); err != nil {
// 		return err
// 	}

// 	// we can ignore this error b/c we might not be running IPFS,
// 	ipfsPeerID, _ := n.IPFSPeerID()
// 	// qriPeerID := n.Identity

// 	for _, p := range peers {
// 		id, err := p.IPFSPeerID()
// 		if err != nil {
// 			fmt.Printf("error decoding base58 peer id: %s\n", err.Error())
// 			continue
// 		}

// 		// skip self
// 		if id == ipfsPeerID {
// 			continue
// 		}

// 		if profile, err := n.Repo.Peers().GetPeer(id); err != nil || profile != nil && profile.Updated.Before(p.Updated) {
// 			if err := n.Repo.Peers().PutPeer(id, p); err != nil {
// 				fmt.Errorf("error putting peer: %s", err.Error())
// 			}
// 		}
// 	}
// 	return nil
// }
