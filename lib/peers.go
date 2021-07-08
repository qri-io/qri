package lib

import (
	"context"
	"fmt"
	"strings"

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
	d dispatcher
}

// Name returns the name of this method group
func (m PeerMethods) Name() string { return "peer" }

// Attributes defines attributes for each method
func (m PeerMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"list":                 {Endpoint: AEPeers, HTTPVerb: "POST"},
		"info":                 {Endpoint: AEPeer, HTTPVerb: "POST"},
		"connect":              {Endpoint: AEConnect, HTTPVerb: "POST"},
		"disconnect":           {Endpoint: AEDisconnect, HTTPVerb: "POST"},
		"connections":          {Endpoint: AEConnections, HTTPVerb: "POST"},
		"connectedqriprofiles": {Endpoint: AEConnectedQriProfiles, HTTPVerb: "POST"},
	}
}

// PeerListParams defines parameters for the List method
type PeerListParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	// Cached == true will return offline peers from the repo
	// as well as online peers, default is to list connected peers only
	Cached bool `json:"cached"`
}

// List lists Peers on the qri network
func (m PeerMethods) List(ctx context.Context, p *PeerListParams) ([]*config.ProfilePod, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "list"), p)
	if res, ok := got.([]*config.ProfilePod); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// PeerInfoParams defines parameters for the Info method
type PeerInfoParams struct {
	Peername  string `json:"peername"`
	ProfileID string `json:"profileID"`
	// Verbose adds network details from the p2p Peerstore
	Verbose bool `json:"verbose"`
}

// Info shows peer profile details
func (m PeerMethods) Info(ctx context.Context, p *PeerInfoParams) (*config.ProfilePod, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "info"), p)
	if res, ok := got.(*config.ProfilePod); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Connect attempts to create a connection with a peer for a given peer.ID
func (m PeerMethods) Connect(ctx context.Context, p *ConnectParamsPod) (*config.ProfilePod, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "connect"), p)
	if res, ok := got.(*config.ProfilePod); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// Disconnect explicitly closes a peer connection
func (m PeerMethods) Disconnect(ctx context.Context, p *ConnectParamsPod) error {
	_, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "disconnect"), p)
	return err
}

// ConnectionsParams defines parameters for the Connections method
type ConnectionsParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Connections lists PeerID's we're currently connected to. If running
// IPFS this will also return connected IPFS nodes
func (m PeerMethods) Connections(ctx context.Context, p *ConnectionsParams) ([]string, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "connections"), p)
	if res, ok := got.([]string); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// ConnectedQriProfiles lists profiles we're currently connected to
func (m PeerMethods) ConnectedQriProfiles(ctx context.Context, p *ConnectionsParams) ([]*config.ProfilePod, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "connectedqriprofiles"), p)
	if res, ok := got.([]*config.ProfilePod); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// ConnectParamsPod defines parameters for defining a connection
// to a peer as plain-old-data
type ConnectParamsPod struct {
	Peername  string `json:"peername"`
	ProfileID string `json:"profileID"`
	NetworkID string `json:"networkID"`
	Multiaddr string `json:"multiaddr"`
}

// NewConnectParamsPod attempts to turn a string into peer connection parameters
func NewConnectParamsPod(s string) *ConnectParamsPod {
	pcpod := &ConnectParamsPod{}

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

func (p ConnectParamsPod) String() string {
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
func (p ConnectParamsPod) Decode() (cp p2p.PeerConnectionParams, err error) {
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

// peerImpl holds the method implementations for PeerMethods
type peerImpl struct{}

// List lists Peers on the qri network
func (peerImpl) List(scope scope, p *PeerListParams) ([]*config.ProfilePod, error) {
	res := []*config.ProfilePod{}

	if scope.Node() == nil || !scope.Node().Online {
		return nil, fmt.Errorf("error: not connected, run `qri connect` in another window")
	}

	var err error

	if p.Limit <= 0 {
		p.Limit = DefaultPageSize
	}

	// requesting user is hardcoded as node owner
	u := scope.ActiveProfile()
	res, err = p2p.ListPeers(scope.Node(), u.ID, p.Offset, p.Limit, !p.Cached)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Info shows peer profile details
func (peerImpl) Info(scope scope, p *PeerInfoParams) (*config.ProfilePod, error) {
	res := config.ProfilePod{}
	// TODO: Move most / all of this to p2p package, perhaps.
	r := scope.Repo()

	var pid profile.ID
	var err error
	if p.ProfileID != "" {
		pid, err = profile.IDB58Decode(p.ProfileID)
		if err != nil {
			return nil, err
		}
	}

	profiles, err := r.Profiles().List()
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	for _, pro := range profiles {
		if pro.ID == pid || pro.Peername == p.Peername {
			if p.Verbose && len(pro.PeerIDs) > 0 {
				// TODO - grab more than just the first peerID
				pinfo := scope.Node().PeerInfo(pro.PeerIDs[0])
				pro.NetworkAddrs = pinfo.Addrs
			}

			prof, err := pro.Encode()
			if err != nil {
				return nil, err
			}
			res = *prof

			connected := scope.Node().ConnectedQriProfiles()

			// If the requested profileID is in the list of connected peers, set Online flag.
			if _, ok := connected[pro.ID]; ok {
				res.Online = true
			}
			// If the requested profileID is myself and I'm Online, set Online flag.
			if peer.ID(pro.ID) == scope.Node().ID && scope.Node().Online {
				res.Online = true
			}
			return &res, nil
		}
	}

	return nil, repo.ErrNotFound
}

// Connect attempts to create a connection with a peer for a given peer.ID
func (peerImpl) Connect(scope scope, p *ConnectParamsPod) (*config.ProfilePod, error) {
	pcp, err := p.Decode()
	if err != nil {
		return nil, err
	}

	prof, err := scope.Node().ConnectToPeer(scope.Context(), pcp)
	if err != nil {
		return nil, err
	}

	pro, err := prof.Encode()
	if err != nil {
		return nil, err
	}

	return pro, nil
}

// Disconnect explicitly closes a peer connection
func (peerImpl) Disconnect(scope scope, p *ConnectParamsPod) error {
	pcp, err := p.Decode()
	if err != nil {
		return err
	}

	return scope.Node().DisconnectFromPeer(scope.Context(), pcp)
}

// Connections lists PeerID's we're currently connected to. If running
// IPFS this will also return connected IPFS nodes
func (peerImpl) Connections(scope scope, p *ConnectionsParams) ([]string, error) {
	// TODO (ramfox): limit and offset not currently used
	// ensure valid limit value
	if p.Limit <= 0 {
		p.Limit = 25
	}
	// ensure valid offset value
	if p.Offset < 0 {
		p.Offset = 0
	}
	return scope.Node().ConnectedPeers(), nil
}

// ConnectedQriProfiles lists profiles we're currently connected to
func (peerImpl) ConnectedQriProfiles(scope scope, p *ConnectionsParams) ([]*config.ProfilePod, error) {
	// TODO (ramfox): offset not currently used
	// ensure valid limit value
	if p.Limit <= 0 {
		p.Limit = 25
	}
	// ensure valid offset value
	if p.Offset < 0 {
		p.Offset = 0
	}

	connected := scope.Node().ConnectedQriProfiles()

	build := make([]*config.ProfilePod, intMin(len(connected), p.Limit))
	for _, pro := range connected {
		build = append(build, pro)
		if len(build) >= p.Limit {
			break
		}
	}
	return build, nil
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
