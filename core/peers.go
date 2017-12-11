package core

import (
	"encoding/json"
	"fmt"
	"net/rpc"

	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

type PeerRequests struct {
	qriNode *p2p.QriNode
	cli     *rpc.Client
}

func NewPeerRequests(node *p2p.QriNode, cli *rpc.Client) *PeerRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both node and client supplied to NewPeerRequests"))
	}

	return &PeerRequests{
		qriNode: node,
		cli:     cli,
	}
}

func (d PeerRequests) CoreRequestsName() string { return "peers" }

func (d *PeerRequests) List(p *ListParams, res *[]*profile.Profile) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.List", p, res)
	}

	r := d.qriNode.Repo
	replies := make([]*profile.Profile, p.Limit)
	i := 0

	user, err := r.Profile()
	if err != nil {
		return err
	}

	ps, err := repo.QueryPeers(r.Peers(), query.Query{})
	if err != nil {
		return fmt.Errorf("error querying peers: %s", err.Error())
	}

	for _, peer := range ps {
		if i >= p.Limit {
			break
		}
		if peer.Id == user.Id {
			continue
		}
		replies[i] = peer
		i++
	}

	*res = replies[:i]
	return nil
}

func (d *PeerRequests) ConnectedPeers(limit *int, peers *[]string) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.ConnectedPeers", limit, peers)
	}

	*peers = d.qriNode.ConnectedPeers()
	return nil
}

func (d *PeerRequests) ConnectToPeer(pid *peer.ID, res *profile.Profile) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.ConnectToPeer", pid, res)
	}

	if err := d.qriNode.ConnectToPeer(*pid); err != nil {
		return fmt.Errorf("error connecting to peer: %s", err.Error())
	}

	profile, err := d.qriNode.Repo.Peers().GetPeer(*pid)
	if err != nil {
		return fmt.Errorf("error getting peer profile: %s", err.Error())
	}

	*res = *profile
	return nil
}

func (d *PeerRequests) Get(p *GetParams, res *profile.Profile) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.Get", p, res)
	}

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

func (d *PeerRequests) GetNamespace(p *NamespaceParams, res *[]*repo.DatasetRef) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.GetNamespace", p, res)
	}

	id, err := peer.IDB58Decode(p.PeerId)
	if err != nil {
		return fmt.Errorf("error decoding peer Id: %s", err.Error())
	}

	profile, err := d.qriNode.Repo.Peers().GetPeer(id)
	if err != nil || profile == nil {
		return err
	}

	r, err := d.qriNode.SendMessage(id, &p2p.Message{
		Phase: p2p.MpRequest,
		Type:  p2p.MtDatasets,
		Payload: &p2p.DatasetsReqParams{
			Limit:  p.Limit,
			Offset: p.Offset,
		},
	})
	if err != nil {
		return fmt.Errorf("error sending message to peer: %s", err.Error())
	}

	data, err := json.Marshal(r.Payload)
	if err != nil {
		return fmt.Errorf("error encoding peer response: %s", err.Error())
	}
	refs := []*repo.DatasetRef{}
	if err := json.Unmarshal(data, &refs); err != nil {
		return fmt.Errorf("error parsing peer response: %s", err.Error())
	}

	*res = refs
	return nil
}
