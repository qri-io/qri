package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo"
)

// RequestDatasetInfo get's qri profile information from a PeerInfo
func (n *QriNode) RequestDatasetInfo(ref *repo.DatasetRef) (*dataset.Dataset, error) {
	id, err := n.Repo.Peers().IPFSPeerID(ref.Peername)
	if err != nil {
		return nil, fmt.Errorf("error getting peer IPFS id: %s", err.Error())
	}

	res, err := n.SendMessage(id, &Message{
		Type:    MtDatasetInfo,
		Phase:   MpRequest,
		Payload: ref,
	})

	if err != nil {
		fmt.Println("send dataset info message error:", err.Error())
		return nil, err
	}

	data, err := json.Marshal(res.Payload)
	if err != nil {
		return nil, err
	}

	ds := &dataset.Dataset{}
	err = json.Unmarshal(data, ds)

	return ds, err
}
