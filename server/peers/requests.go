package peers

import (
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

func NewRequests(r repo.Repo, node *p2p.QriNode) *Requests {
	return &Requests{
		repo:    r,
		qriNode: node,
	}
}

type Requests struct {
	repo    repo.Repo
	qriNode *p2p.QriNode
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

func (d *Requests) List(p *ListParams, res *[]*profile.Profile) error {
	replies := make([]*profile.Profile, p.Limit)
	i := 0

	ps, err := repo.QueryPeers(d.repo.Peers(), query.Query{})
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	for _, peer := range ps {
		if i >= p.Limit {
			break
		}
		replies[i] = peer
		i++
	}

	*res = replies[:i]
	return nil
}

type GetParams struct {
	Username string
	Name     string
	Hash     string
}

func (d *Requests) Get(p *GetParams, res *profile.Profile) error {
	// TODO - restore
	// peers, err := d.repo.Peers()
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return err
	// }

	// for name, repo := range peers {
	// 	if p.Hash == name ||
	// 		p.Username == repo.Profile.Username {
	// 		*res = *repo.Profile
	// 	}
	// }

	// if res != nil {
	// 	return nil
	// }

	// TODO - ErrNotFound plz
	return fmt.Errorf("Not Found")
}

type NamespaceParams struct {
	PeerId string
	Limit  int
	Offset int
}

func (d *Requests) GetNamespace(p *NamespaceParams, res *[]*repo.DatasetRef) error {
	id, err := peer.IDB58Decode(p.PeerId)
	if err != nil {
		return err
	}

	fmt.Println(id.Pretty())

	profile, err := d.repo.Peers().GetPeer(id)
	if err != nil || profile == nil {
		return err
	}

	fmt.Println(profile.Username)

	r, err := d.qriNode.SendMessage(id, &p2p.Message{
		Phase: p2p.MpRequest,
		Type:  p2p.MtDatasets,
		Payload: &p2p.DatasetsReqParams{
			Limit:  p.Limit,
			Offset: p.Offset,
		},
	})
	if err != nil {
		return err
	}

	data, err := json.Marshal(r.Payload)
	if err != nil {
		return err
	}
	refs := []*repo.DatasetRef{}
	if err := json.Unmarshal(data, &refs); err != nil {
		return err
	}

	*res = refs
	return nil
}
