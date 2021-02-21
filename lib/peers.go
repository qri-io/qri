package lib

import (
	"context"
	"fmt"
	"strings"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"

	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// PeerMethods extends a lib.Instance with business logic for peer-to-peer
// interaction
type PeerMethods struct {
	inst *Instance
}

// CoreRequestsName implements the Requets interface
func (m PeerMethods) CoreRequestsName() string { return "peers" }

// NewPeerMethods creates a p2p Methods pointer
func NewPeerMethods(inst *Instance) *PeerMethods {
	return &PeerMethods{
		inst: inst,
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
func (m *PeerMethods) List(p *PeerListParams, res *[]*config.ProfilePod) (err error) {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("PeerMethods.List", p, res))
	}
	if m.inst.node == nil || !m.inst.node.Online {
		return fmt.Errorf("error: not connected, run `qri connect` in another window")
	}

	if p.Limit <= 0 {
		p.Limit = DefaultPageSize
	}

	// requesting user is hardcoded as node owner
	u := m.inst.repo.Profiles().Owner()
	*res, err = p2p.ListPeers(m.inst.node, u.ID, p.Offset, p.Limit, !p.Cached)
	return err
}

// ConnectedIPFSPeers lists PeerID's we're currently connected to. If running
// IPFS this will also return connected IPFS nodes
func (m *PeerMethods) ConnectedIPFSPeers(limit *int, peers *[]string) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("PeerMethods.ConnectedIPFSPeers", limit, peers))
	}

	*peers = m.inst.node.ConnectedPeers()
	return nil
}

// ConnectedQriProfiles lists profiles we're currently connected to
func (m *PeerMethods) ConnectedQriProfiles(limit *int, peers *[]*config.ProfilePod) (err error) {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("PeerMethods.ConnectedQriProfiles", limit, peers))
	}

	connected := m.inst.node.ConnectedQriProfiles()

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
func (m *PeerMethods) ConnectToPeer(p *PeerConnectionParamsPod, res *config.ProfilePod) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("PeerMethods.ConnectToPeer", p, res))
	}
	ctx := context.TODO()

	pcp, err := p.Decode()
	if err != nil {
		return err
	}

	prof, err := m.inst.node.ConnectToPeer(ctx, pcp)
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
func (m *PeerMethods) DisconnectFromPeer(p *PeerConnectionParamsPod, res *bool) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("PeerMethods.DisconnectFromPeer", p, res))
	}
	ctx := context.TODO()

	pcp, err := p.Decode()
	if err != nil {
		return err
	}

	if err := m.inst.node.DisconnectFromPeer(ctx, pcp); err != nil {
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
func (m *PeerMethods) Info(p *PeerInfoParams, res *config.ProfilePod) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("PeerMethods.Info", p, res))
	}

	// TODO: Move most / all of this to p2p package, perhaps.
	r := m.inst.repo

	profiles, err := r.Profiles().List()
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	for _, pro := range profiles {
		if pro.ID == p.ProfileID || pro.Peername == p.Peername {
			if p.Verbose && len(pro.PeerIDs) > 0 {
				// TODO - grab more than just the first peerID
				pinfo := m.inst.node.PeerInfo(pro.PeerIDs[0])
				pro.NetworkAddrs = pinfo.Addrs
			}

			prof, err := pro.Encode()
			if err != nil {
				return err
			}
			*res = *prof

			connected := m.inst.node.ConnectedQriProfiles()

			// If the requested profileID is in the list of connected peers, set Online flag.
			if _, ok := connected[pro.ID]; ok {
				res.Online = true
			}
			// If the requested profileID is myself and I'm Online, set Online flag.
			if peer.ID(pro.ID) == m.inst.node.ID && m.inst.node.Online {
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
func (m *PeerMethods) GetReferences(p *PeerRefsParams, res *[]reporef.DatasetRef) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("PeerMethods.GetReferences", p, res))
	}
	ctx := context.TODO()

	id, err := peer.IDB58Decode(p.PeerID)
	if err != nil {
		return fmt.Errorf("error decoding peer Id: %s", err.Error())
	}

	refs, err := m.inst.node.RequestDatasetsList(ctx, id, p2p.DatasetsListParams{
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
