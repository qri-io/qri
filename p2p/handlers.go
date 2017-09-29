package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/core/search"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
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
	fmt.Println("added peer:", pi.ID.Pretty())
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
	names, err := n.Repo.Namespace(p.Limit, p.Offset)
	if err != nil {
		fmt.Println("repo names error:", err)
		return nil
	}

	replies := make([]*repo.DatasetRef, p.Limit)
	i := 0
	for name, path := range names {
		if i >= p.Limit {
			break
		}
		ds, err := dsfs.LoadDataset(n.Store, path)
		if err != nil {
			fmt.Println("error loading dataset at path:", path)
			return nil
		}
		replies[i] = &repo.DatasetRef{
			Name:    name,
			Path:    path,
			Dataset: ds,
		}
		i++
	}

	replies = replies[:i]
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
	ds := []*repo.DatasetRef{}
	if err := json.Unmarshal(data, &ds); err != nil {
		return err
	}
	// pinfo.Namespace = ns
	// peers[pi.ID.Pretty()] = pinfo
	// fmt.Println("added peer dataset info:", pi.ID.Pretty())
	// fmt.Println(ds)

	return n.Repo.Cache().PutDatasets(ds)
}

func (qn *QriNode) Search(terms string, limit, offset int) (res []*repo.DatasetRef, err error) {
	responses, err := qn.BroadcastMessage(&Message{
		Phase: MpRequest,
		Type:  MtSearch,
		Payload: &SearchParams{
			Query:  terms,
			Limit:  limit,
			Offset: offset,
		},
	})
	if err != nil {
		return nil, err
	}

	// fmt.Println(responses)
	datasets := []*repo.DatasetRef{}

	for _, r := range responses {
		data, err := json.Marshal(r.Payload)
		if err != nil {
			return datasets, err
		}
		ds := []*repo.DatasetRef{}
		if err := json.Unmarshal(data, &ds); err != nil {
			return datasets, err
		}
		datasets = append(datasets, ds...)
	}

	return datasets, nil
}

type SearchParams struct {
	Query  string
	Limit  int
	Offset int
}

func (n *QriNode) handleSearchRequest(r *Message) *Message {
	data, err := json.Marshal(r.Payload)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	p := &SearchParams{}
	if err := json.Unmarshal(data, p); err != nil {
		fmt.Println("unmarshal search request error:", err.Error())
		return nil
	}

	results, err := search.Search(n.Repo, n.Store, search.NewDatasetQuery(p.Query, p.Limit, p.Offset))
	return &Message{
		Phase:   MpResponse,
		Type:    MtSearch,
		Payload: results,
	}
}

func (n *QriNode) handleSearchResponse(pi pstore.PeerInfo, m *Message) error {
	return fmt.Errorf("not yet finished")
}
