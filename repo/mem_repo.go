package repo

import (
	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo/profile"
)

// MemRepo is an in-memory implementation of the Repo interface
type MemRepo struct {
	store cafs.Filestore
	graph map[string]*dsgraph.Node
	MemDatasets
	*MemNamestore
	*MemQueryLog
	MemChangeRequests
	profile   *profile.Profile
	peers     Peers
	cache     MemDatasets
	analytics analytics.Analytics
}

// NewMemRepo creates a new in-memory repository
func NewMemRepo(p *profile.Profile, store cafs.Filestore, ps Peers, a analytics.Analytics) (Repo, error) {
	return &MemRepo{
		store:             store,
		MemDatasets:       MemDatasets{},
		MemNamestore:      &MemNamestore{},
		MemQueryLog:       &MemQueryLog{},
		MemChangeRequests: MemChangeRequests{},
		profile:           p,
		peers:             ps,
		analytics:         a,
		cache:             MemDatasets{},
	}, nil
}

// Store returns the underlying cafs.Filestore for this repo
func (r *MemRepo) Store() cafs.Filestore {
	return r.store
}

// Graph gives the graph of objects in this repo
func (r *MemRepo) Graph() (map[string]*dsgraph.Node, error) {
	return RepoGraph(r)
}

// Profile returns the peer profile for this repository
func (r *MemRepo) Profile() (*profile.Profile, error) {
	return r.profile, nil
}

// SaveProfile updates this repo's profile
func (r *MemRepo) SaveProfile(p *profile.Profile) error {
	r.profile = p
	return nil
}

// Peers gives this repo's Peer interface implementation
func (r *MemRepo) Peers() Peers {
	return r.peers
}

// Cache gives this repo's ephemeral cache of Datasets
func (r *MemRepo) Cache() Datasets {
	return r.cache
}

// Analytics returns this repo's analytics store
func (r *MemRepo) Analytics() analytics.Analytics {
	return r.analytics
}
