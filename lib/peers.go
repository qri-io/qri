package lib

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"

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

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *PeerListParams) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &PeerListParams{}
	}

	params := *p

	if r.FormValue("limit") != "" && r.FormValue("offset") != "" {
		params.Limit = util.ReqParamInt(r, "limit", 0)
		params.Offset = util.ReqParamInt(r, "offset", 0)
	} else {
		lp := &ListParams{}
		if err := lp.UnmarshalFromRequest(r); err != nil {
			return err
		}
		params.Limit = lp.Limit
		params.Offset = lp.Offset
	}

	if !params.Cached {
		params.Cached = r.FormValue("cached") == "true"
	}

	*p = params
	return nil
}

// List lists Peers on the qri network
func (m *PeerMethods) List(ctx context.Context, p *PeerListParams) ([]*config.ProfilePod, error) {
	res := []*config.ProfilePod{}
	if m.inst.http != nil {
		qvars := map[string]string{
			"limit":  strconv.Itoa(p.Limit),
			"offset": strconv.Itoa(p.Offset),
			"cached": strconv.FormatBool(p.Cached),
		}
		err := m.inst.http.CallMethod(ctx, AEPeers, http.MethodGet, qvars, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	if m.inst.node == nil || !m.inst.node.Online {
		return nil, fmt.Errorf("error: not connected, run `qri connect` in another window")
	}

	var err error

	if p.Limit <= 0 {
		p.Limit = DefaultPageSize
	}

	// requesting user is hardcoded as node owner
	u := m.inst.repo.Profiles().Owner()
	res, err = p2p.ListPeers(m.inst.node, u.ID, p.Offset, p.Limit, !p.Cached)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ConnectedIPFSPeers lists PeerID's we're currently connected to. If running
// IPFS this will also return connected IPFS nodes
func (m *PeerMethods) ConnectedIPFSPeers(ctx context.Context, limit *int) ([]string, error) {
	res := []string{}
	if m.inst.http != nil {
		err := m.inst.http.CallMethodRaw(ctx, AEConnections, http.MethodGet, nil, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	res = m.inst.node.ConnectedPeers()
	return res, nil
}

// ConnectedQriProfiles lists profiles we're currently connected to
func (m *PeerMethods) ConnectedQriProfiles(ctx context.Context, limit *int) ([]*config.ProfilePod, error) {
	res := []*config.ProfilePod{}
	if m.inst.http != nil {
		qvars := map[string]string{
			"limit": strconv.Itoa(*limit),
		}
		err := m.inst.http.CallMethod(ctx, AEConnectionsQri, http.MethodGet, qvars, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	connected := m.inst.node.ConnectedQriProfiles()

	build := make([]*config.ProfilePod, intMin(len(connected), *limit))
	for _, p := range connected {
		build = append(build, p)
		if len(build) >= *limit {
			break
		}
	}
	return build, nil
}

// PeerConnectionParamsPod defines parameters for defining a connection
// to a peer as plain-old-data
type PeerConnectionParamsPod struct {
	Peername  string
	ProfileID string
	NetworkID string
	Multiaddr string
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *PeerConnectionParamsPod) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &PeerConnectionParamsPod{}
	}

	if r.FormValue("path") != "" {
		*p = *NewPeerConnectionParamsPod(r.FormValue("path"))
	}

	return nil
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

func (p PeerConnectionParamsPod) String() string {
	if p.Peername != "" {
		return p.Peername
	}
	if p.ProfileID != "" {
		return p.ProfileID
	}
	if p.Multiaddr != "" {
		return p.Multiaddr
	}
	if p.NetworkID != "" {
		return p.NetworkID
	}
	return ""
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
func (m *PeerMethods) ConnectToPeer(ctx context.Context, p *PeerConnectionParamsPod) (*config.ProfilePod, error) {
	res := &config.ProfilePod{}
	if m.inst.http != nil {
		err := m.inst.http.Call(ctx, AEConnect, p, res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	pcp, err := p.Decode()
	if err != nil {
		return nil, err
	}

	prof, err := m.inst.node.ConnectToPeer(ctx, pcp)
	if err != nil {
		return nil, err
	}

	pro, err := prof.Encode()
	if err != nil {
		return nil, err
	}

	return pro, nil
}

// DisconnectFromPeer explicitly closes a peer connection
func (m *PeerMethods) DisconnectFromPeer(ctx context.Context, p *PeerConnectionParamsPod) error {
	if m.inst.http != nil {
		route := AEConnect.WithSuffix(p.String())
		err := m.inst.http.CallMethod(ctx, route, http.MethodDelete, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	pcp, err := p.Decode()
	if err != nil {
		return err
	}

	return m.inst.node.DisconnectFromPeer(ctx, pcp)
}

// PeerInfoParams defines parameters for the Info method
type PeerInfoParams struct {
	Peername  string
	ProfileID profile.ID
	// Verbose adds network details from the p2p Peerstore
	Verbose bool
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *PeerInfoParams) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &PeerInfoParams{}
	}

	if r.FormValue("profile") != "" {
		pid, err := profile.IDB58Decode(r.FormValue("profile"))
		if err != nil {
			p.Peername = r.FormValue("profile")
		} else {
			p.ProfileID = pid
		}
	}

	if !p.Verbose {
		p.Verbose = r.FormValue("verbose") == "true"
	}

	return nil
}

// Info shows peer profile details
func (m *PeerMethods) Info(ctx context.Context, p *PeerInfoParams) (*config.ProfilePod, error) {
	res := config.ProfilePod{}
	if m.inst.http != nil {
		route := AEPeers.WithSuffix(p.ProfileID.String())
		if p.ProfileID.String() == "" {
			route = AEPeers.WithSuffix(p.Peername)
		}
		qvars := map[string]string{
			"verbose": strconv.FormatBool(p.Verbose),
		}
		err := m.inst.http.CallMethod(ctx, route, http.MethodGet, qvars, &res)
		if err != nil {
			return nil, err
		}
		return &res, nil
	}

	// TODO: Move most / all of this to p2p package, perhaps.
	r := m.inst.repo

	profiles, err := r.Profiles().List()
	if err != nil {
		log.Debug(err.Error())
		return nil, err
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
				return nil, err
			}
			res = *prof

			connected := m.inst.node.ConnectedQriProfiles()

			// If the requested profileID is in the list of connected peers, set Online flag.
			if _, ok := connected[pro.ID]; ok {
				res.Online = true
			}
			// If the requested profileID is myself and I'm Online, set Online flag.
			if peer.ID(pro.ID) == m.inst.node.ID && m.inst.node.Online {
				res.Online = true
			}
			return &res, nil
		}
	}

	return nil, repo.ErrNotFound
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
