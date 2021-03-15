package p2p

// TODO (ramfox): relies on old `depQriProtocolID`
// Should have its own protocol & protobuf & not rely on the Message struct

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qri-io/qri/base"
	reporef "github.com/qri-io/qri/repo/ref"

	peer "github.com/libp2p/go-libp2p-core/peer"
)

// MtDatasets is a dataset list message
const MtDatasets = MsgType("list_datasets")

// listMax is the highest number of entries a list request should return
const listMax = 30

// DatasetsListParams encapsulates options for requesting datasets
type DatasetsListParams struct {
	Term   string
	Limit  int
	Offset int
}

// RequestDatasetsList gets a list of a peer's datasets
func (n *QriNode) RequestDatasetsList(ctx context.Context, pid peer.ID, p DatasetsListParams) ([]reporef.DatasetRef, error) {
	log.Debugf("%s RequestDatasetList: %s", n.ID, pid)

	if pid == n.ID {
		// requesting self isn't a network operation
		return n.Repo.References(p.Offset, p.Limit)
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

	s, err := n.host.NewStream(ctx, pid, depQriProtocolID)
	if err != nil {
		return nil, fmt.Errorf("error opening stream: %s", err.Error())
	}
	defer s.Close()

	ws := WrapStream(s)
	if err := ws.sendMessage(req); err != nil {
		return nil, err
	}

	res, err := ws.receiveMessage()
	if err != nil {
		return nil, err
	}

	ref := []reporef.DatasetRef{}
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

		refs, err := base.ListDatasets(context.TODO(), n.Repo, dlp.Term, "", dlp.Offset, dlp.Limit, false, true, false)
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
