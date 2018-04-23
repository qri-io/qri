package p2p

import (
	"encoding/json"
	"time"

	"github.com/qri-io/qri/repo/profile"

	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

// MtProfile is a peer info message
const MtProfile = MsgType("profile")

// RequestProfile get's qri profile information on a peer ID
func (n *QriNode) RequestProfile(pid peer.ID) (*profile.Profile, error) {
	log.Debugf("%s RequestProfile: %s", n.ID, pid)

	if pid == n.ID {
		// if we request ourself... well that's not a network call at all :)
		return n.Repo.Profile()
	}

	// Get this repo's profile information
	data, err := n.profileBytes()
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	replies := make(chan Message)
	msg := NewMessage(n.ID, MtProfile, data)

	if err := n.SendMessage(msg, replies, pid); err != nil {
		log.Debugf("send profile message error: %s", err.Error())
		return nil, err
	}

	res := <-replies
	log.Debugf("profile response for message: %s", res.ID)

	cp := &profile.CodingProfile{}
	if err := json.Unmarshal(res.Body, cp); err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	pro := &profile.Profile{}
	if err := pro.Decode(cp); err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	if err := n.Repo.Profiles().PutProfile(pro); err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	return pro, nil
}

func (n *QriNode) handleProfile(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true

	pro := &profile.Profile{}
	if err := json.Unmarshal(msg.Body, pro); err != nil {
		log.Debug(err.Error())
		return
	}

	pro.Updated = time.Now()

	data, err := n.profileBytes()
	if err != nil {
		log.Debug(err.Error())
		return
	}

	if err := ws.sendMessage(msg.Update(data)); err != nil {
		log.Debugf("error sending peer info message: %s", err.Error())
	}

	return
}

func (n *QriNode) profileBytes() ([]byte, error) {
	p, err := n.Repo.Profile()
	if err != nil {
		log.Debugf("error getting repo profile: %s\n", err.Error())
		return nil, err
	}
	cp, err := p.Encode()
	if err != nil {
		log.Debugf("error encoding repo profile: %s\n", err.Error())
		return nil, err
	}

	return json.Marshal(cp)
}
