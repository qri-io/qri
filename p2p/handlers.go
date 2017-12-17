package p2p

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore/query"
	"time"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
)

func (n *QriNode) handlePingRequest(r *Message) *Message {
	return &Message{
		Phase: MpResponse,
		Type:  MtPing,
	}
}

func (n *QriNode) handlePeerInfoRequest(r *Message) *Message {
	go func(r *Message) error {
		data, err := json.Marshal(r.Payload)
		if err != nil {
			return nil
		}
		p := &profile.Profile{}
		if err := json.Unmarshal(data, p); err != nil {
			return err
		}

		pid, err := p.PeerID()
		if err != nil {
			return fmt.Errorf("error decoding base58 peer id: %s", err.Error())
		}

		p.Updated = time.Now()
		n.log.Infof("adding peer: %s\n", pid.Pretty())
		return n.Repo.Peers().PutPeer(pid, p)
	}(r)

	p, err := n.Repo.Profile()
	if err != nil {
		n.log.Infof("error getting repo profile: %s\n", err.Error())
		return nil
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

	n.log.Info("adding peer:", pi.ID.Pretty())
	return n.Repo.Peers().PutPeer(pi.ID, p)
}

// PeersReqParams outlines params for requesting peers
type PeersReqParams struct {
	Limit  int
	Offset int
}

func (n *QriNode) handlePeersRequest(r *Message) *Message {
	data, err := json.Marshal(r.Payload)
	if err != nil {
		n.log.Info(err.Error())
		return nil
	}
	p := &PeersReqParams{}
	if err := json.Unmarshal(data, p); err != nil {
		n.log.Info("unmarshal peers request error:", err.Error())
		return nil
	}

	profiles, err := repo.QueryPeers(n.Repo.Peers(), query.Query{
		Limit:  p.Limit,
		Offset: p.Offset,
	})

	if err != nil {
		n.log.Info("error getting peer profiles:", err.Error())
		return nil
	}

	return &Message{
		Type:    MtPeers,
		Phase:   MpResponse,
		Payload: profiles,
	}
}

func (n *QriNode) handlePeersResponse(r *Message) error {
	data, err := json.Marshal(r.Payload)
	if err != nil {
		return err
	}
	peers := []*profile.Profile{}
	if err := json.Unmarshal(data, &peers); err != nil {
		return err
	}

	for _, p := range peers {
		id, err := p.PeerID()
		if err != nil {
			fmt.Printf("error decoding base58 peer id: %s\n", err.Error())
			continue
		}
		if profile, err := n.Repo.Peers().GetPeer(id); err != nil || profile != nil && profile.Updated.Before(p.Updated) {
			if err := n.Repo.Peers().PutPeer(id, p); err != nil {
				fmt.Errorf("error putting peer: %s", err.Error())
			}
		}
	}
	return nil
}

// DatasetsReqParams encapsulates options for requesting datasets
type DatasetsReqParams struct {
	Query  string
	Limit  int
	Offset int
}

func (n *QriNode) handleDatasetsRequest(r *Message) *Message {
	data, err := json.Marshal(r.Payload)
	if err != nil {
		n.log.Info(err.Error())
		return nil
	}
	p := &DatasetsReqParams{}
	if err := json.Unmarshal(data, p); err != nil {
		n.log.Info("unmarshal dataset request error:", err.Error())
		return nil
	}

	if p.Limit == 0 {
		p.Limit = 50
	}
	refs, err := n.Repo.Namespace(p.Limit, p.Offset)
	if err != nil {
		n.log.Info("repo names error:", err)
		return nil
	}

	// replies := make([]*repo.DatasetRef, p.Limit)
	// i := 0
	for i, ref := range refs {
		if i >= p.Limit {
			break
		}
		ds, err := dsfs.LoadDataset(n.Repo.Store(), ref.Path)
		if err != nil {
			n.log.Info("error loading dataset at path:", ref.Path)
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

// Search broadcasts a search request to all connected peers, aggregating results
func (n *QriNode) Search(terms string, limit, offset int) (res []*repo.DatasetRef, err error) {
	responses, err := n.BroadcastMessage(&Message{
		Phase: MpRequest,
		Type:  MtSearch,
		Payload: &repo.SearchParams{
			Q:      terms,
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

func (n *QriNode) handleSearchRequest(r *Message) *Message {
	n.log.Info("handling search request")
	data, err := json.Marshal(r.Payload)
	if err != nil {
		n.log.Info(err.Error())
		return nil
	}
	p := &repo.SearchParams{}
	if err := json.Unmarshal(data, p); err != nil {
		n.log.Info("unmarshal search request error:", err.Error())
		return nil
	}

	// results, err := search.Search(n.Repo, n.Store, search.NewDatasetQuery(p.Query, p.Limit, p.Offset))
	if s, ok := n.Repo.(repo.Searchable); ok {
		results, err := s.Search(*p)
		if err != nil {
			n.log.Info("search error:", err.Error())
			return nil
		}
		return &Message{
			Phase:   MpResponse,
			Type:    MtSearch,
			Payload: results,
		}
	}

	return &Message{
		Phase:   MpError,
		Type:    MtSearch,
		Payload: fmt.Errorf("repo doesn't support search"),
	}
}

func (n *QriNode) handleSearchResponse(pi pstore.PeerInfo, m *Message) error {
	return fmt.Errorf("not yet finished")
}
