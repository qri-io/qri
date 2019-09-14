package p2p

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qri-io/qri/repo"
)

// MtResolveDatasetRef resolves a dataset reference
const MtResolveDatasetRef = MsgType("resolve_dataset_ref")

// ResolveDatasetRef completes a dataset reference
func (n *QriNode) ResolveDatasetRef(ctx context.Context, ref *repo.DatasetRef) (err error) {
	log.Debugf("%s ResolveDatasetRef %s", n.ID, ref)

	if !n.Online {
		return ErrNotConnected
	}

	pids := n.ClosestConnectedQriPeers(ref.ProfileID, 15)
	if len(pids) == 0 {
		return fmt.Errorf("no connected peers")
	}

	replies := make(chan Message)
	req, err := NewJSONBodyMessage(n.ID, MtResolveDatasetRef, ref)
	req = req.WithHeaders("phase", "request")
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	for _, pid := range pids {
		if err := n.SendMessage(ctx, req, replies, pid); err != nil {
			log.Debugf("%s err: %s", pid, err.Error())
			continue
		}

		res := <-replies
		dsr := repo.DatasetRef{}
		if err := json.Unmarshal(res.Body, &dsr); err == nil {
			if dsr.Path != "" {
				*ref = dsr
				break
			}
		}
	}

	return nil
}

func (n *QriNode) handleResolveDatasetRef(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true

	switch msg.Header("phase") {
	case "request":
		dsr := &repo.DatasetRef{}
		if err := json.Unmarshal(msg.Body, dsr); err != nil {
			log.Debug(err.Error())
			return
		}
		res := msg

		if err := repo.CanonicalizeDatasetRef(n.Repo, dsr); err == nil && dsr.Complete() {
			res, err = msg.UpdateJSON(dsr)
			if err != nil {
				log.Debug(err.Error())
				return
			}
		} else {
			res, err = msg.UpdateJSON(nil)
			if err != nil {
				log.Error(err.Error())
				return
			}
		}

		res = res.WithHeaders("phase", "response")
		if err := ws.sendMessage(res); err != nil {
			log.Debug(err.Error())
			return
		}
	}

	return
}
