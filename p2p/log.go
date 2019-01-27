package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

const (
	// MtDatasetLog gets log of a dataset
	MtDatasetLog = MsgType("dataset_log")
	// NumPeersToContact is the number of peers to contact with messages
	NumPeersToContact = 15
)

// DatasetLogRequest encapsulates options for requesting dataset history
type DatasetLogRequest struct {
	Ref    repo.DatasetRef
	Limit  int
	Offset int
}

// DatasetLogResponse encapsulates option for responding to a dataset history request
type DatasetLogResponse struct {
	History []repo.DatasetRef
	Err     error
}

// RequestDatasetLog gets the log information of Peer's dataset
func (n *QriNode) RequestDatasetLog(ref repo.DatasetRef, limit, offset int) ([]repo.DatasetRef, error) {

	// get a list of peers to whom we will send the request
	pids := n.ClosestConnectedQriPeers(ref.ProfileID, NumPeersToContact)
	if len(pids) == 0 {
		return nil, fmt.Errorf("no connected peers")
	}

	messages := make(chan Message)
	body := DatasetLogRequest{
		Ref:    ref,
		Limit:  limit,
		Offset: offset,
	}

	req, err := NewJSONBodyMessage(n.ID, MtDatasetLog, body)
	req = req.WithHeaders("phase", "request")
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	for _, pid := range pids {
		if err = n.SendMessage(req, messages, pid); err != nil {
			log.Debugf("%s err: %s", pid, err.Error())
			continue
		}
		msg := <-messages
		logResponse := DatasetLogResponse{}
		if err = json.Unmarshal(msg.Body, &logResponse); err == nil {
			// Expect any peer who responds with a non-empty history list to have the
			// authoritative answer. Return as soon as such a response is received.
			if logResponse.Err != nil {
				log.Debugf("%s err: %s", pid, err.Error())
				continue
			}
			if len(logResponse.History) != 0 {
				return logResponse.History, nil
			}
		}
	}
	return nil, fmt.Errorf("unable to locate dataset log for %s", ref)
}

func (n *QriNode) handleDatasetLog(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true

	switch msg.Header("phase") {
	case "request":

		req := DatasetLogRequest{}
		if err := json.Unmarshal(msg.Body, &req); err != nil {
			log.Debug(err.Error())
			return
		}

		ref := req.Ref
		limit := req.Limit
		offset := req.Offset

		history := []repo.DatasetRef{}

		err := repo.CanonicalizeDatasetRef(n.Repo, &ref)
		if err == repo.ErrNotFound {
			// non-local dataset, return early
			sendDatasetLogReply(ws, msg, history, nil)
			return
		}

		for {
			dataset, err := dsfs.LoadDataset(n.Repo.Store(), ref.Path)
			if err != nil {
				sendDatasetLogReply(ws, msg, history, err)
				return
			}
			ref.Dataset = dataset.Encode()

			offset--
			if offset > 0 {
				ref.Path = ref.Dataset.PreviousPath
				continue
			}

			history = append(history, ref)

			limit--
			if limit > 0 && ref.Dataset.PreviousPath != "" {
				ref.Path = ref.Dataset.PreviousPath
				continue
			}
			break
		}

		sendDatasetLogReply(ws, msg, history, nil)
	}
	return
}

func sendDatasetLogReply(ws *WrappedStream, msg Message, history []repo.DatasetRef, err error) {
	response := DatasetLogResponse{}
	response.History = history
	updated, err := msg.UpdateJSON(response)
	if err != nil {
		log.Debug(err.Error())
		return
	}

	updated = updated.WithHeaders("phase", "response")
	if err := ws.sendMessage(updated); err != nil {
		log.Debug(err.Error())
	}
}
