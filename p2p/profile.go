package p2p

import (
	"encoding/json"
	"fmt"
	// "time"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo/profile"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
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
	req := NewMessage(n.ID, MtProfile, data)
	req = req.WithHeaders("phase", "request")

	if err := n.SendMessage(req, replies, pid); err != nil {
		log.Debugf("send profile message error: %s", err.Error())
		return nil, err
	}

	res := <-replies

	pp := &config.ProfilePod{}
	if err := json.Unmarshal(res.Body, pp); err != nil {
		log.Debug(err.Error())
		return nil, err
	}
	pp.PeerIDs = []string{
		fmt.Sprintf("/ipfs/%s", pid.Pretty()),
	}

	pro := &profile.Profile{}
	if err := pro.Decode(pp); err != nil {
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
	// hangup = false

	switch msg.Header("phase") {
	case "request":

		pp := &config.ProfilePod{}
		if err := json.Unmarshal(msg.Body, pp); err != nil {
			log.Debug(err.Error())
			return
		}

		pro := &profile.Profile{}
		if err := pro.Decode(pp); err != nil {
			log.Debug(err.Error())
			return
		}

		if err := n.Repo.Profiles().PutProfile(pro); err != nil {
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
	}
	return
}

func (n *QriNode) profileBytes() ([]byte, error) {
	p, err := n.Repo.Profile()
	if err != nil {
		log.Debugf("error getting repo profile: %s\n", err.Error())
		return nil, err
	}
	pod, err := p.Encode()
	if err != nil {
		log.Debugf("error encoding repo profile: %s\n", err.Error())
		return nil, err
	}

	return json.Marshal(pod)
}
