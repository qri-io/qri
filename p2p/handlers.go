package p2p

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	// "github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo/peer_repo"
	"github.com/qri-io/qri/repo/profile"
)

func (n *QriNode) handlePeerInfoRequest(r *Message) *Message {
	p, err := n.repo.Profile()
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
	peers, err := n.repo.Peers()
	if err != nil {
		return err
	}
	pinfo := peers[pi.ID.Pretty()]
	if pinfo == nil {
		pinfo = &peer_repo.Repo{}
	}

	data, err := json.Marshal(r.Payload)
	if err != nil {
		return err
	}
	p := &profile.Profile{}
	if err := json.Unmarshal(data, p); err != nil {
		return err
	}
	pinfo.Profile = p
	peers[pi.ID.Pretty()] = pinfo
	// fmt.Println("added peer:", pi.ID.Pretty())
	return n.repo.SavePeers(peers)
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

	// replies := make([]*dataset.DatasetRef, p.Limit)
	// TODO - generate a sorted copy of keys, iterate through, respecting
	// limit & offset
	ns, err := n.repo.Namespace()
	if err != nil {
		fmt.Println("repo namespace error:", err.Error())
		return nil
	}

	// for name, key := range ns {
	// 	if i >= p.Limit {
	// 		break
	// 	}
	// 	// ds, err := dataset.LoadDataset(n., key)
	// 	// if err != nil {
	// 	// 	fmt.Println("error loading path:", key)
	// 	// 	return err
	// 	// }
	// 	replies[i] = &dataset.DatasetRef{
	// 		Name: name,
	// 		Path: key,
	// 		// Dataset: ds,
	// 	}
	// 	i++
	// }

	return &Message{
		Type:    MtDatasets,
		Phase:   MpResponse,
		Payload: ns,
	}
}

func (n *QriNode) handleDatasetsResponse(pi pstore.PeerInfo, r *Message) error {
	peers, err := n.repo.Peers()
	if err != nil {
		return err
	}
	pinfo := peers[pi.ID.Pretty()]
	if pinfo == nil {
		pinfo = &peer_repo.Repo{}
	}

	data, err := json.Marshal(r.Payload)
	if err != nil {
		return err
	}
	ns := map[string]datastore.Key{}
	if err := json.Unmarshal(data, &ns); err != nil {
		return err
	}
	pinfo.Namespace = ns
	peers[pi.ID.Pretty()] = pinfo
	fmt.Println("added peer dataset info:", pi.ID.Pretty())
	return n.repo.SavePeers(peers)
}
