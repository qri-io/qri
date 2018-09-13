package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

// MtDatasetInfo gets info on a dataset
const MtDatasetInfo = MsgType("dataset_info")

// RequestDataset fetches info about a dataset from qri peers
// It's expected the local peer has attempted to canonicalize the reference
// before sending to the network
// ref is used as an outparam, populating with data on success
func (n *QriNode) RequestDataset(ref *repo.DatasetRef) (err error) {
	log.Debugf("%s RequestDataset %s", n.ID, ref)

	// if peer ID is *our* peer.ID check for local dataset
	// note that data may be on another machine, so this can still fail back to a
	// network request
	if ref.ProfileID != "" {
		if pro, err := n.Repo.Profile(); err == nil && pro.ID == ref.ProfileID {
			ds, err := dsfs.LoadDataset(n.Repo.Store(), datastore.NewKey(ref.Path))
			if err != nil {
				return err
			}
			ref.Dataset = ds.Encode()
			return nil
		}
	}

	pids := n.ClosestConnectedPeers(ref.ProfileID, 15)
	if len(pids) == 0 {
		// TODO - start checking peerstore peers?
		// something else should probably be trying to establish
		// rolling connections
		return fmt.Errorf("no connected peers")
	}

	replies := make(chan Message)
	req, err := NewJSONBodyMessage(n.ID, MtDatasetInfo, ref)
	req = req.WithHeaders("phase", "request")
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	for _, pid := range pids {

		if err := n.SendMessage(req, replies, pid); err != nil {
			log.Debugf("%s err: %s", pid, err.Error())
			continue
			// return err
		}

		res := <-replies
		dsr := repo.DatasetRef{}
		if err := json.Unmarshal(res.Body, &dsr); err == nil {
			if dsr.Dataset != nil {
				*ref = dsr
				break
			}
		}
	}

	return nil
}

func (n *QriNode) handleDataset(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true

	switch msg.Header("phase") {
	case "request":
		dsr := repo.DatasetRef{}
		if err := json.Unmarshal(msg.Body, &dsr); err != nil {
			log.Debug(err.Error())
			return
		}
		res := msg

		if err := repo.CanonicalizeDatasetRef(n.Repo, &dsr); err == nil {
			if ref, err := n.Repo.GetRef(dsr); err == nil {
				ds, e := dsfs.LoadDataset(n.Repo.Store(), datastore.NewKey(ref.Path))
				if e != nil {
					log.Debug(err.Error())
					return
				}
				ref.Dataset = ds.Encode()

				res, err = msg.UpdateJSON(ref)
				if err != nil {
					log.Debug(err.Error())
					return
				}
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
