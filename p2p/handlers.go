package p2p

import (
	"encoding/json"
	"fmt"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo/profile"
)

func (n *QriNode) handlePeerInfoRequest(r *Message) *Message {
	p, err := n.Repo.Profile()
	if err != nil {
		fmt.Println(err.Error())
	}
	return &Message{
		Type:    MtPeerInfo,
		Phase:   MpResponse,
		Payload: p,
	}
}

func (n *QriNode) handleProfileResponse(pi pstore.PeerInfo, r *Message) error {
	// peers, err := n.Repo.Peers()
	// if err != nil {
	// 	return err
	// }
	// pinfo := peers[pi.ID.Pretty()]
	// if pinfo == nil {
	// 	pinfo = &profile.Profile{}
	// }

	data, err := json.Marshal(r.Payload)
	if err != nil {
		return err
	}
	p := &profile.Profile{}
	if err := json.Unmarshal(data, p); err != nil {
		return err
	}
	// pinfo.Profile = p
	// peers[pi.ID.Pretty()] = pinfo
	// fmt.Println("added peer:", pi.ID.Pretty())
	return n.Repo.Peers().PutPeer(pi.ID, p)
}

type DatasetsReqParams struct {
	Query  string
	Limit  int
	Offset int
}

func (n *QriNode) handleDatasetsRequest(r *Message) *Message {
	data, err := json.Marshal(r.Payload)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	p := &DatasetsReqParams{}
	if err := json.Unmarshal(data, p); err != nil {
		fmt.Println("unmarshal dataset request error:", err.Error())
		return nil
	}

	if p.Limit == 0 {
		p.Limit = 50
	}
	names, err := n.Repo.Names(p.Limit, p.Offset)
	if err != nil {
		fmt.Println("repo names error:", err)
		return nil
	}

	replies := make([]*dataset.DatasetRef, p.Limit)
	i := 0
	for name, path := range names {
		if i >= p.Limit {
			break
		}
		ds, err := dataset.LoadDataset(n.Store, path)
		if err != nil {
			fmt.Println("error loading dataset at path:", path)
			return nil
		}
		replies[i] = &dataset.DatasetRef{
			Name:    name,
			Path:    path,
			Dataset: ds,
		}
		i++
	}

	return &Message{
		Type:    MtDatasets,
		Phase:   MpResponse,
		Payload: replies,
	}
}

func (n *QriNode) handleDatasetsResponse(pi pstore.PeerInfo, r *Message) error {
	// peers, err := n.Repo.Peers()
	// if err != nil {
	// 	return err
	// }
	// pinfo := peers[pi.ID.Pretty()]
	// if pinfo == nil {
	// 	pinfo = &profile.Profile{}
	// }

	data, err := json.Marshal(r.Payload)
	if err != nil {
		return err
	}
	ds := []*dataset.DatasetRef{}
	if err := json.Unmarshal(data, &ds); err != nil {
		return err
	}
	// pinfo.Namespace = ns
	// peers[pi.ID.Pretty()] = pinfo
	// fmt.Println("added peer dataset info:", pi.ID.Pretty())
	return n.Repo.Cache().PutDatasets(ds)
}
