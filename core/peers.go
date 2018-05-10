package core

import (
	"context"
	"fmt"
	"net/rpc"
	"strings"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	ma "gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
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

// PeerListParams defines parameters for the List method
type PeerListParams struct {
	Limit, Offset int
	// Cached == true will return offline peers from the repo
	// as well as online peers, default is to list connected peers only
	Cached bool
}

// List lists Peers on the qri network
func (d *PeerRequests) List(p *PeerListParams, res *[]*config.ProfilePod) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.List", p, res)
	}

	r := d.qriNode.Repo
	user, err := r.Profile()
	if err != nil {
		return err
	}

	peers := make([]*config.ProfilePod, p.Limit)
	online := []*config.ProfilePod{}
	if err := d.ConnectedQriProfiles(&p.Limit, &online); err != nil {
		return err
	}

	if !p.Cached {
		*res = online
		return nil
	}

	ps, err := r.Profiles().List()
	if err != nil {
		return fmt.Errorf("error listing peers: %s", err.Error())
	}

	if len(ps) == 0 {
		*res = []*config.ProfilePod{}
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

		// TODO - this is dumb use a map
		for _, olp := range online {
			if pro.ID.String() == olp.ID {
				pro.Online = true
			}
		}

		peers[i], err = pro.Encode()
		if err != nil {
			return err
		}

		i++
	}

	*res = peers
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

// ConnectedQriProfiles lists profiles we're currently connected to
func (d *PeerRequests) ConnectedQriProfiles(limit *int, peers *[]*config.ProfilePod) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.ConnectedQriProfiles", limit, peers)
	}

	parsed := []*config.ProfilePod{}
	for _, p := range d.qriNode.ConnectedQriProfiles() {
		// pro, err := p.Encode()
		// if err != nil {
		// 	return err
		// }
		parsed = append(parsed, p)
	}

	*peers = parsed
	return nil
}

// PeerConnectionParamsPod defines parameters for defining a connection
// to a peer as plain-old-data
type PeerConnectionParamsPod struct {
	Peername  string
	ProfileID string
	NetworkID string
	Multiaddr string
}

// NewPeerConnectionParamsPod attempts to turn a string into peer connection parameters
func NewPeerConnectionParamsPod(s string) *PeerConnectionParamsPod {
	pcpod := &PeerConnectionParamsPod{}

	if maddr, err := ma.NewMultiaddr(s); err == nil {
		pcpod.Multiaddr = maddr.String()
	} else if strings.HasPrefix(s, "/ipfs/") {
		pcpod.NetworkID = s
	} else if len(s) == 46 && strings.HasPrefix(s, "Qm") {
		pcpod.ProfileID = s
	} else {
		pcpod.Peername = s
	}

	return pcpod
}

// Decode turns plain-old-data into it's rich types
func (p PeerConnectionParamsPod) Decode() (cp p2p.PeerConnectionParams, err error) {
	cp.Peername = p.Peername

	if p.NetworkID != "" {
		id := strings.TrimPrefix(p.NetworkID, "/ipfs/")
		if len(id) == len(p.NetworkID) {
			err = fmt.Errorf("network IDs must have a network prefix (eg. /ipfs/)")
			return
		}
		if cp.PeerID, err = peer.IDB58Decode(id); err != nil {
			err = fmt.Errorf("invalid networkID: %s", err.Error())
			return
		}
	}

	if p.ProfileID != "" {
		if cp.ProfileID, err = profile.IDB58Decode(p.ProfileID); err != nil {
			err = fmt.Errorf("invalid profileID: %s", err.Error())
			return
		}
	}

	if p.Multiaddr != "" {
		if cp.Multiaddr, err = ma.NewMultiaddr(p.Multiaddr); err != nil {
			err = fmt.Errorf("invalid multiaddr: %s", err.Error())
		}
	}

	return
}

// ConnectToPeer attempts to create a connection with a peer for a given peer.ID
func (d *PeerRequests) ConnectToPeer(p *PeerConnectionParamsPod, res *config.ProfilePod) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.ConnectToPeer", p, res)
	}

	pcp, err := p.Decode()
	if err != nil {
		return err
	}

	prof, err := d.qriNode.ConnectToPeer(context.Background(), pcp)
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

// DisconnectFromPeer explicitly closes a peer connection
func (d *PeerRequests) DisconnectFromPeer(p *PeerConnectionParamsPod, res *bool) error {
	if d.cli != nil {
		return d.cli.Call("PeerRequests.DisconnectFromPeer", p, res)
	}

	pcp, err := p.Decode()
	if err != nil {
		return err
	}

	if err := d.qriNode.DisconnectFromPeer(context.Background(), pcp); err != nil {
		return err
	}

	*res = true
	return nil
}

// PeerInfoParams defines parameters for the Info method
type PeerInfoParams struct {
	Peername  string
	ProfileID profile.ID
}

// Info shows peer profile details
func (d *PeerRequests) Info(p *PeerInfoParams, res *config.ProfilePod) error {
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
