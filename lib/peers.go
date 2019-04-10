package lib

import (
	"context"
	"fmt"
	"net/rpc"
	"strings"

	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"

	ma "gx/ipfs/QmT4U94DnD8FRfqr21obWY32HLM5VExccPKMjQHofeYqr9/go-multiaddr"
	peer "gx/ipfs/QmTRhk7cgjUf2gfQ3p2M9KPECNZEW9XUrmHcFCgog4cPgB/go-libp2p-peer"
)

// PeerMethods access & work with peers on peer-2-peer networks
type PeerMethods interface {
	Methods
	List(p *PeerListParams, res *[]*config.ProfilePod) error
	ConnectedIPFSPeers(limit *int, peers *[]string) error
	ConnectedQriProfiles(limit *int, peers *[]*config.ProfilePod) error
	ConnectToPeer(p *PeerConnectionParamsPod, res *config.ProfilePod) error
	DisconnectFromPeer(p *PeerConnectionParamsPod, res *bool) error
	Info(p *PeerInfoParams, res *config.ProfilePod) error
	GetReferences(p *PeerRefsParams, res *[]repo.DatasetRef) error
}

// NewPeerMethods creates a peerMethods pointer from either a
// qri Node or an rpc.Client
func NewPeerMethods(inst Instance) PeerMethods {
	return peerMethods{
		qriNode: inst.Node(),
		cli:     inst.RPC(),
	}
}

// peerMethods encapsulates business logic for methods
// relating to peer-to-peer interaction
type peerMethods struct {
	qriNode *p2p.QriNode
	cli     *rpc.Client
}

// MethodsKind implements the Requets interface
func (d peerMethods) MethodsKind() string { return "PeerMethods" }

// PeerListParams defines parameters for the List method
type PeerListParams struct {
	Limit, Offset int
	// Cached == true will return offline peers from the repo
	// as well as online peers, default is to list connected peers only
	Cached bool
}

// List lists Peers on the qri network
func (d peerMethods) List(p *PeerListParams, res *[]*config.ProfilePod) (err error) {
	if d.cli != nil {
		return d.cli.Call("PeerMethods.List", p, res)
	}
	if d.qriNode == nil {
		return fmt.Errorf("error: not connected, run `qri connect` in another window")
	}

	*res, err = actions.ListPeers(d.qriNode, p.Limit, p.Offset, !p.Cached)
	return err
}

// ConnectedIPFSPeers lists PeerID's we're currently connected to. If running
// IPFS this will also return connected IPFS nodes
func (d peerMethods) ConnectedIPFSPeers(limit *int, peers *[]string) error {
	if d.cli != nil {
		return d.cli.Call("PeerMethods.ConnectedIPFSPeers", limit, peers)
	}

	*peers = d.qriNode.ConnectedPeers()
	return nil
}

// ConnectedQriProfiles lists profiles we're currently connected to
func (d peerMethods) ConnectedQriProfiles(limit *int, peers *[]*config.ProfilePod) (err error) {
	if d.cli != nil {
		return d.cli.Call("PeerMethods.ConnectedQriProfiles", limit, peers)
	}

	connected, err := actions.ConnectedQriProfiles(d.qriNode)
	if err != nil {
		return err
	}

	build := make([]*config.ProfilePod, intMin(len(connected), *limit))
	for _, p := range connected {
		build = append(build, p)
		if len(build) >= *limit {
			break
		}
	}
	*peers = build
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

	if strings.HasPrefix(s, "/ipfs/") {
		pcpod.NetworkID = s
	} else if maddr, err := ma.NewMultiaddr(s); err == nil {
		pcpod.Multiaddr = maddr.String()
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
func (d peerMethods) ConnectToPeer(p *PeerConnectionParamsPod, res *config.ProfilePod) error {
	if d.cli != nil {
		return d.cli.Call("PeerMethods.ConnectToPeer", p, res)
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
func (d peerMethods) DisconnectFromPeer(p *PeerConnectionParamsPod, res *bool) error {
	if d.cli != nil {
		return d.cli.Call("PeerMethods.DisconnectFromPeer", p, res)
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
	// Verbose adds network details from the p2p Peerstore
	Verbose bool
}

// Info shows peer profile details
func (d peerMethods) Info(p *PeerInfoParams, res *config.ProfilePod) error {
	if d.cli != nil {
		return d.cli.Call("PeerMethods.Info", p, res)
	}

	// TODO: Move most / all of this to actions package, perhaps.
	r := d.qriNode.Repo

	profiles, err := r.Profiles().List()
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	for _, pro := range profiles {
		if pro.ID == p.ProfileID || pro.Peername == p.Peername {
			if p.Verbose && len(pro.PeerIDs) > 0 {
				// TODO - grab more than just the first peerID
				pinfo := d.qriNode.PeerInfo(pro.PeerIDs[0])
				pro.NetworkAddrs = pinfo.Addrs
			}

			prof, err := pro.Encode()
			*res = *prof

			connected, err := actions.ConnectedQriProfiles(d.qriNode)
			if err != nil {
				return err
			}
			// If the requested profileID is in the list of connected peers, set Online flag.
			if _, ok := connected[pro.ID]; ok {
				res.Online = true
			}
			// If the requested profileID is myself and I'm Online, set Online flag.
			if peer.ID(pro.ID) == d.qriNode.ID && d.qriNode.Online {
				res.Online = true
			}
			return nil

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
func (d peerMethods) GetReferences(p *PeerRefsParams, res *[]repo.DatasetRef) error {
	if d.cli != nil {
		return d.cli.Call("PeerMethods.GetReferences", p, res)
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

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
