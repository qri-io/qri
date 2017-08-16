package p2p

import (
	"encoding/json"
	"fmt"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/qri-io/qri/repo/peer_repo"
	"github.com/qri-io/qri/repo/profile"
)

func (n *QriNode) handleProfileRequest(r *Message) *Message {
	p, err := n.repo.Profile()
	if err != nil {
		fmt.Println(err.Error())
	}
	return &Message{
		Type:    MtProfile,
		Phase:   MpResponse,
		Payload: p,
	}
}

func (n *QriNode) handleProfileResponse(pi pstore.PeerInfo, r *Message) error {
	peers, err := n.repo.Peers()
	if err != nil {
		return err
	}
	pinfo := peers[pi.ID.Pretty()]
	if pinfo == nil {
		pinfo = &peer_repo.Repo{}
	}

	data, err := json.Marshal(r.Payload)
	if err != nil {
		return err
	}
	p := &profile.Profile{}
	if err := json.Unmarshal(data, p); err != nil {
		return err
	}
	pinfo.Profile = p
	peers[pi.ID.Pretty()] = pinfo

	return n.repo.SavePeers(peers)
}
