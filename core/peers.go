package core

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

// PeerRequests encapsulates business logic for methods
// relating to peer-to-peer interaction
type PeerRequests struct {
	qriNode *p2p.QriNode
	cli     *rpc.Client
}

// CoreRequestsName implements the Requets interface
func (d PeerRequests) CoreRequestsName() string { return "peers" }

// NewPeerRequests creates a PeerRequests pointer from either a
// qri Node or an rpc.Client
func NewPeerRequests(node *p2p.QriNode, cli *rpc.Client) *PeerRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both node and client supplied to NewPeerRequests"))
	}

	return &PeerRequests{
		qriNode: node,
		cli:     cli,
	}
}

// List lists Peers on the qri network
func (d *PeerRequests) List(p *ListParams, res *[]*profile.CodingProfile) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.List", p, res)
	}

	r := d.qriNode.Repo
	replies := make([]*profile.CodingProfile, p.Limit)

	user, err := r.Profile()
	if err != nil {
		return err
	}

	ps, err := r.Profiles().List()
	if err != nil {
		return fmt.Errorf("error listing peers: %s", err.Error())
	}

	if len(ps) == 0 {
		*res = []*profile.CodingProfile{}
		return nil
	}

	i := 0
	for _, pro := range ps {
		if i >= p.Limit {
			break
		}
		if pro == nil || pro.ID == user.ID {
			continue
		}
		replies[i], err = pro.Encode()
		if err != nil {
			return err
		}

		i++
	}

	*res = replies[:i]
	return nil
}

// ConnectedIPFSPeers lists PeerID's we're currently connected to. If running
// IPFS this will also return connected IPFS nodes
func (d *PeerRequests) ConnectedIPFSPeers(limit *int, peers *[]string) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.ConnectedIPFSPeers", limit, peers)
	}

	*peers = d.qriNode.ConnectedPeers()
	return nil
}

// // Peer is a quick proxy for profile.Profile that plays
// // nice with encoding/gob
// type Peer struct {
// 	ID       string
// 	IPFSID   string
// 	Peername string
// 	Name     string
// }

// ConnectedQriProfiles lists profiles we're currently connected to
func (d *PeerRequests) ConnectedQriProfiles(limit *int, peers *[]*profile.CodingProfile) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.ConnectedQriProfiles", limit, peers)
	}

	parsed := []*profile.CodingProfile{}
	for _, p := range d.qriNode.ConnectedQriProfiles() {
		pro, err := p.Encode()
		if err != nil {
			return err
		}
		parsed = append(parsed, pro)
	}

	*peers = parsed
	return nil
}

// ConnectToPeer attempts to create a connection with a peer for a given peer.ID
func (d *PeerRequests) ConnectToPeer(b58pid *string, res *profile.CodingProfile) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.ConnectToPeer", b58pid, res)
	}

	pid, err := peer.IDB58Decode(*b58pid)
	if err != nil {
		return err
	}

	if err := d.qriNode.ConnectToPeer(pid); err != nil {
		return nil
	}

	prof, err := d.qriNode.Repo.Profiles().PeerProfile(pid)
	if err != nil {
		return err
	}
	pro, err := prof.Encode()
	if err != nil {
		return err
	}

	*res = *pro
	return nil
}

// PeerInfoParams defines parameters for the Info method
type PeerInfoParams struct {
	Peername  string
	ProfileID profile.ID
}

// Info shows peer profile details
func (d *PeerRequests) Info(p *PeerInfoParams, res *profile.CodingProfile) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.Info", p, res)
	}

	r := d.qriNode.Repo

	profiles, err := r.Profiles().List()
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	for _, pro := range profiles {
		if pro.ID == p.ProfileID || pro.Peername == p.Peername {
			prof, err := pro.Encode()
			*res = *prof
			return err
		}
	}

	return repo.ErrNotFound
}

// PeerRefsParams defines params for the GetReferences method
type PeerRefsParams struct {
	PeerID string
	Limit  int
	Offset int
}

// GetReferences lists a peer's named datasets
func (d *PeerRequests) GetReferences(p *PeerRefsParams, res *[]repo.DatasetRef) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.GetReferences", p, res)
	}

	id, err := peer.IDB58Decode(p.PeerID)
	if err != nil {
		return fmt.Errorf("error decoding peer Id: %s", err.Error())
	}

	// profile, err := d.qriNode.Repo.Profiles().GetProfile(id)
	// if err != nil || profile == nil {
	// 	return err
	// }

	refs, err := d.qriNode.RequestDatasetsList(id, p2p.DatasetsListParams{
		Limit:  p.Limit,
		Offset: p.Offset,
	})

	*res = refs
	return err
}
