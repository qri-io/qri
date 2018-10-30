package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/repo"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
)

// MtDatasets is a dataset list message
const MtDatasets = MsgType("list_datasets")

// listMax is the highest number of entries a list request should return
const listMax = 30

// DatasetsListParams encapsulates options for requesting datasets
type DatasetsListParams struct {
	Limit  int
	Offset int
}

// RequestDatasetsList gets a list of a peer's datasets
func (n *QriNode) RequestDatasetsList(pid peer.ID, p DatasetsListParams) ([]repo.DatasetRef, error) {
	log.Debugf("%s RequestDatasetList: %s", n.ID, pid)

	if pid == n.ID {
		// requesting self isn't a network operation
		return n.Repo.References(p.Limit, p.Offset)
	}

	if !n.Online {
		return nil, fmt.Errorf("not connected to p2p network")
	}

	req, err := NewJSONBodyMessage(n.ID, MtDatasets, p)
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	req = req.WithHeaders("phase", "request")

	replies := make(chan Message)
	err = n.SendMessage(req, replies, pid)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("send dataset info message error: %s", err.Error())
	}

	res := <-replies
	ref := []repo.DatasetRef{}
	err = json.Unmarshal(res.Body, &ref)
	return ref, err
}

func (n *QriNode) handleDatasetsList(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true
	switch msg.Header("phase") {
	case "request":
		dlp := DatasetsListParams{}
		if err := json.Unmarshal(msg.Body, &dlp); err != nil {
			log.Debugf("%s %s", n.ID, err.Error())
			return
		}

		if dlp.Limit == 0 || dlp.Limit > listMax {
			dlp.Limit = listMax
		}

		refs, err := base.ListDatasets(n.Repo, dlp.Limit, dlp.Offset, false, true)
		if err != nil {
			log.Error(err)
			return
		}

		reply, err := msg.UpdateJSON(refs)
		reply = reply.WithHeaders("phase", "response")
		if err := ws.sendMessage(reply); err != nil {
			log.Debug(err.Error())
			return
		}
	}

	return
}
