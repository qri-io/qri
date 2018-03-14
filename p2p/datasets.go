package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/qri/repo"
)

// RequestDatasetsList gets a list of a peer's datasets
func (n *QriNode) RequestDatasetsList(peername string) ([]repo.DatasetRef, error) {
	id, err := n.Repo.Peers().IPFSPeerID(peername)
	if err != nil {
		return nil, fmt.Errorf("error getting peer IPFS id: %s", err.Error())
	}

	res, err := n.SendMessage(id, &Message{
		Type:    MtDatasets,
		Phase:   MpRequest,
		Payload: nil,
	})

	if err != nil {
		fmt.Println("send dataset info message error:", err.Error())
		return nil, err
	}

	data, err := json.Marshal(res.Payload)
	if err != nil {
		log.Debugf(err.Error())
		return nil, err
	}

	ref := []repo.DatasetRef{}
	err = json.Unmarshal(data, &ref)
	return ref, err
}

// RequestDatasetInfo get's qri profile information from a PeerInfo
func (n *QriNode) RequestDatasetInfo(ref *repo.DatasetRef) (*repo.DatasetRef, error) {
	id, err := n.Repo.Peers().IPFSPeerID(ref.Peername)
	if err != nil {
		log.Debugf("error getting peer IPFS id: %s", err.Error())
		return nil, err
	}

	res, err := n.SendMessage(id, &Message{
		Type:    MtDatasetInfo,
		Phase:   MpRequest,
		Payload: ref,
	})

	if err != nil {
		log.Debugf("send dataset info message error:", err.Error())
		return nil, err
	}

	data, err := json.Marshal(res.Payload)
	if err != nil {
		return nil, err
	}

	resref := &repo.DatasetRef{}
	err = json.Unmarshal(data, resref)

	return resref, err
}

// RequestDatasetLog gets the log information of Peer's dataset
func (n *QriNode) RequestDatasetLog(ref repo.DatasetRef) (*[]repo.DatasetRef, error) {
	id, err := n.Repo.Peers().IPFSPeerID(ref.Peername)
	if err != nil {
		return nil, fmt.Errorf("error getting peer IPFS id: %s", err.Error())
	}
	res, err := n.SendMessage(id, &Message{
		Type:    MtDatasetLog,
		Phase:   MpRequest,
		Payload: ref,
	})
	if err != nil {
		log.Debugf("send dataset log message error: %s", err.Error())
		return nil, err
	}

	data, err := json.Marshal(res.Payload)
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	resref := []repo.DatasetRef{}
	err = json.Unmarshal(data, &resref)
	if len(resref) == 0 && err != nil {
		err = fmt.Errorf("no log found")
	}

	return &resref, err
}
