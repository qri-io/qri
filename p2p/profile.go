package p2p

import (
	"encoding/json"
	"time"

	"github.com/qri-io/qri/repo/profile"

	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
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
	log.Debug(res)

	pro := &profile.Profile{}
	if err := json.Unmarshal(res.Body, pro); err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	if err := n.Repo.Peers().PutPeer(pid, pro); err != nil {
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
	pid, err := pro.IPFSPeerID()
	if err != nil {
		log.Debug(err.Error())
		return
	}
	pro.Updated = time.Now()

	log.Debugf("adding peer: %s", pid.Pretty())
	if err := n.Repo.Peers().PutPeer(pid, pro); err != nil {
		log.Debug(err.Error())
		return
	}

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

	if addrs, err := n.IPFSListenAddresses(); err == nil {
		p.Addresses = addrs
	}

	return json.Marshal(p)
}
