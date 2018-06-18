package repo

import (
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regclient"
)

// MemRepo is an in-memory implementation of the Repo interface
type MemRepo struct {
	*MemRefstore
	*MemEventLog

	store        cafs.Filestore
	graph        map[string]*dsgraph.Node
	refCache     *MemRefstore
	selectedRefs []DatasetRef

	profile  *profile.Profile
	profiles profile.Store
	registry *regclient.Client
}

// NewMemRepo creates a new in-memory repository
func NewMemRepo(p *profile.Profile, store cafs.Filestore, ps profile.Store, rc *regclient.Client) (Repo, error) {
	return &MemRepo{
		store:       store,
		MemRefstore: &MemRefstore{},
		MemEventLog: &MemEventLog{},
		refCache:    &MemRefstore{},
		profile:     p,
		profiles:    ps,
		registry:    rc,
	}, nil
}

// Store returns the underlying cafs.Filestore for this repo
func (r *MemRepo) Store() cafs.Filestore {
	return r.store
}

// PrivateKey returns this repo's private key
func (r *MemRepo) PrivateKey() crypto.PrivKey {
	if r.profile == nil {
		return nil
	}
	return r.profile.PrivKey
}

// RefCache gives access to the ephemeral Refstore
func (r *MemRepo) RefCache() Refstore {
	return r.refCache
}

// Graph gives the graph of objects in this repo
func (r *MemRepo) Graph() (map[string]*dsgraph.Node, error) {
	return Graph(r)
}

// Profile returns the peer profile for this repository
func (r *MemRepo) Profile() (*profile.Profile, error) {
	return r.profile, nil
}

// SetProfile updates this repo's profile
func (r *MemRepo) SetProfile(p *profile.Profile) error {
	r.profile = p
	return nil
}

// SetSelectedRefs sets the current reference selection
func (r *MemRepo) SetSelectedRefs(sel []DatasetRef) error {
	r.selectedRefs = sel
	return nil
}

// SelectedRefs gives the current reference selection
func (r *MemRepo) SelectedRefs() ([]DatasetRef, error) {
	return r.selectedRefs, nil
}

// Profiles gives this repo's Peer interface implementation
func (r *MemRepo) Profiles() profile.Store {
	return r.profiles
}

// Registry returns a client for interacting with a federated registry if one exists, otherwise nil
func (r *MemRepo) Registry() *regclient.Client {
	return r.registry
}
