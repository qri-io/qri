package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/repo"
)

// MtLogDiff gets info on a dataset
const MtLogDiff = MsgType("log_diff")

// RequestLogDiff fetches info about a dataset from qri peers
// It's expected the local peer has attempted to canonicalize the reference
// before sending to the network
func (n *QriNode) RequestLogDiff(ref *repo.DatasetRef) (ldr base.LogDiffResult, err error) {
	log.Debugf("%s RequestLogDiff %s", n.ID, ref)

	p, err := n.ConnectToPeer(n.ctx, PeerConnectionParams{
		Peername:  ref.Peername,
		ProfileID: ref.ProfileID,
	})

	if err != nil {
		err = fmt.Errorf("coudn't connection to peer: %s", err.Error())
		return
	}

	// TODO - deal with max limit / offset / pagination issuez
	rLog, err := base.DatasetLog(n.Repo, *ref, 10000, 0, false)
	if err != nil {
		return
	}

	replies := make(chan Message)
	req, err := NewJSONBodyMessage(n.ID, MtLogDiff, rLog)
	req = req.WithHeaders("phase", "request")
	if err != nil {
		log.Debug(err.Error())
		return
	}

	for _, pid := range p.PeerIDs {
		if err = n.SendMessage(req, replies, pid); err != nil {
			log.Debugf("%s err: %s", pid, err.Error())
			continue
		}

		res := <-replies
		ldr = base.LogDiffResult{}
		if err = json.Unmarshal(res.Body, &ldr); err == nil {
			break
		}
	}

	return
}

func (n *QriNode) handleLogDiff(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true

	switch msg.Header("phase") {
	case "request":
		remoteLog := []repo.DatasetRef{}
		if err := json.Unmarshal(msg.Body, &remoteLog); err != nil {
			log.Debug(err.Error())
			return
		}

		res := msg
		ldr, err := base.LogDiff(n.Repo, remoteLog)
		if err != nil {
			log.Error(err)
			return
		}

		res, err = msg.UpdateJSON(ldr)
		if err != nil {
			log.Error(err)
			return
		}

		res = res.WithHeaders("phase", "response")
		if err := ws.sendMessage(res); err != nil {
			log.Debug(err.Error())
			return
		}
	}

	return
}
