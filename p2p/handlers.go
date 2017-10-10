package p2p

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/qri-io/dataset/dsfs"
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

	// ignore any id property in case peers a lying jerks
	p.Id = pi.ID.Pretty()
	p.Updated = time.Now()

	fmt.Println("adding peer:", pi.ID.Pretty())
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
	refs, err := n.Repo.Namespace(p.Limit, p.Offset)
	if err != nil {
		fmt.Println("repo names error:", err)
		return nil
	}

	// replies := make([]*repo.DatasetRef, p.Limit)
	// i := 0
	for i, ref := range refs {
		if i >= p.Limit {
			break
		}
		ds, err := dsfs.LoadDataset(n.Store, ref.Path)
		if err != nil {
			fmt.Println("error loading dataset at path:", ref.Path)
			return nil
		}
		refs[i].Dataset = ds
		// i++
	}

	// replies = replies[:i]
	return &Message{
		Type:    MtDatasets,
		Phase:   MpResponse,
		Payload: refs,
	}
}

func (n *QriNode) handleDatasetsResponse(pi pstore.PeerInfo, r *Message) error {
	data, err := json.Marshal(r.Payload)
	if err != nil {
		return err
	}
	ds := []*repo.DatasetRef{}
	if err := json.Unmarshal(data, &ds); err != nil {
		return err
	}

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
	fmt.Println("handling search request")
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

	// results, err := search.Search(n.Repo, n.Store, search.NewDatasetQuery(p.Query, p.Limit, p.Offset))
	if s, ok := n.Repo.(repo.Searchable); ok {
		results, err := s.Search(p.Query)
		if err != nil {
			fmt.Println("search error:", err.Error())
			return nil
		}
		return &Message{
			Phase:   MpResponse,
			Type:    MtSearch,
			Payload: results,
		}
	} else {
		// TODO - repo doesn't support search
	}

	return nil
}

func (n *QriNode) handleSearchResponse(pi pstore.PeerInfo, m *Message) error {
	return fmt.Errorf("not yet finished")
}
