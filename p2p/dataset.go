package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo"

	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// MtDatasetInfo gets info on a dataset
const MtDatasetInfo = MsgType("dataset_info")

// RequestDataset fetches info about a dataset from qri peers
// It's expected the local peer has attempted to canonicalize the reference
// before sending to the network
// ref is used as an outparam, populating with data on success
func (n *QriNode) RequestDataset(ref *repo.DatasetRef) (err error) {
	log.Debugf("%s RequestDataset %s", n.ID, ref)

	if ref.Path == "" {
		return fmt.Errorf("path is required")
	}

	// if peer ID is *our* peer.ID check for local dataset
	// note that data may be on another machine, so this can still fail back to a
	// network request
	if ref.PeerID != "" {
		if pro, err := n.Repo.Profile(); err == nil && pro.ID == ref.PeerID {
			if ds, err := n.Repo.GetDataset(datastore.NewKey(ref.Path)); err == nil {
				ref.Dataset = ds
				return nil
			}
		}
	}

	var pid peer.ID
	if ref.PeerID != "" {
		if id, err := peer.IDB58Decode(ref.PeerID); err == nil {
			pid = id
		}
	}

	pids := n.ClosestConnectedPeers(pid, 15)
	if len(pids) == 0 {
		log.Debug(err.Error())

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
			log.Debug(err.Error())
			return err
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

				if ds, err := n.Repo.GetDataset(datastore.NewKey(ref.Path)); err == nil {
					ref.Dataset = ds
				}

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
