package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

const (
	// MtDatasetLog gets log of a dataset
	MtDatasetLog = MsgType("dataset_log")
)

// DatasetLogRequestBody encapsulates options for requesting dataset history
type DatasetLogRequestBody struct {
	Ref    repo.DatasetRef
	Limit  int
	Offset int
}

// DatasetLogResponseBody encapsulates option for responding to a dataset history request
type DatasetLogResponseBody struct {
	History []repo.DatasetRef
	Err     error
}

// RequestDatasetLog gets the log information of Peer's dataset
func (n *QriNode) RequestDatasetLog(ref repo.DatasetRef, limit, offset int) ([]repo.DatasetRef, error) {

	// get a list of peers to whom we will send the request
	pids := n.ClosestConnectedPeers(ref.ProfileID, 15)
	if len(pids) == 0 {
		return nil, fmt.Errorf("no connected peers")
	}

	// create a channel on which to send the requests and receive the responses
	replies := make(chan Message)

	// create body of message:
	body := DatasetLogRequestBody{
		Ref:    ref,
		Limit:  limit,
		Offset: offset,
	}

	// create the request
	req, err := NewJSONBodyMessage(n.ID, MtDatasetLog, body)
	req = req.WithHeaders("phase", "request")
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	// iterate through the peer ids
	for _, pid := range pids {

		if err = n.SendMessage(req, replies, pid); err != nil {
			log.Debugf("%s err: %s", pid, err.Error())
			continue
		}

		res := <-replies
		dlrb := DatasetLogResponseBody{}
		if err = json.Unmarshal(res.Body, &dlrb); err == nil {
			if dlrb.Err != nil {
				log.Debugf("%s err: %s", pid, err.Error())
				continue
			}
			if len(dlrb.History) != 0 {
				return dlrb.History, nil
			}
		}
	}
	return nil, fmt.Errorf("unable to locate dataset log for %s", ref)
}

func (n *QriNode) handleDatasetLog(ws *WrappedStream, req Message) (hangup bool) {
	hangup = true

	switch req.Header("phase") {
	case "request":

		resBody := DatasetLogRequestBody{}
		if err := json.Unmarshal(req.Body, &resBody); err != nil {
			log.Debug(err.Error())
			return
		}

		ref := resBody.Ref
		limit := resBody.Limit
		offset := resBody.Offset

		local := true

		err := n.ResolveDatasetRef(&ref)
		if err != nil {
			local = false
		}

		reqBody := DatasetLogResponseBody{}
		history := []repo.DatasetRef{}

		if local {
			for {
				dataset, err := dsfs.LoadDataset(n.Repo.Store(), datastore.NewKey(ref.Path))
				if err != nil {
					reqBody.Err = err
					break
				}
				ref.Dataset = dataset.Encode()

				offset--
				if offset > 0 {
					continue
				}

				history = append(history, ref)

				limit--
				if limit == 0 || ref.Dataset.PreviousPath == "" {
					break
				}
				ref.Path = ref.Dataset.PreviousPath
			}
		}

		reqBody.History = history
		res, err := req.UpdateJSON(reqBody)
		if err != nil {
			log.Debug(err.Error())
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
