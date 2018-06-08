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
	pk       crypto.PrivKey
	store    cafs.Filestore
	graph    map[string]*dsgraph.Node
	refCache *MemRefstore
	*MemRefstore
	*MemEventLog
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

// SetPrivateKey sets this repos's internal private key reference
func (r *MemRepo) SetPrivateKey(pk crypto.PrivKey) error {
	r.pk = pk
	return nil
}

// PrivateKey returns this repo's private key
func (r *MemRepo) PrivateKey() crypto.PrivKey {
	return r.pk
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

// Profiles gives this repo's Peer interface implementation
func (r *MemRepo) Profiles() profile.Store {
	return r.profiles
}

// Registry returns a client for interacting with a federated registry if one exists, otherwise nil
func (r *MemRepo) Registry() *regclient.Client {
	return r.registry
}
