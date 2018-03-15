package repo

import (
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo/profile"
)

// MemRepo is an in-memory implementation of the Repo interface
type MemRepo struct {
	pk    crypto.PrivKey
	store cafs.Filestore
	graph map[string]*dsgraph.Node
	MemDatasets
	*MemRefstore
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
		MemDatasets:       NewMemDatasets(store),
		MemRefstore:       &MemRefstore{},
		MemQueryLog:       &MemQueryLog{},
		MemChangeRequests: MemChangeRequests{},
		profile:           p,
		peers:             ps,
		analytics:         a,
		cache:             NewMemDatasets(store),
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

// Graph gives the graph of objects in this repo
func (r *MemRepo) Graph() (map[string]*dsgraph.Node, error) {
	return Graph(r)
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

// CreateDataset initializes a dataset from a dataset pointer and data file
func (r *MemRepo) CreateDataset(name string, ds *dataset.Dataset, data cafs.File, pin bool) (path datastore.Key, err error) {
	path, err = dsfs.CreateDataset(r.store, ds, data, r.pk, pin)
	if err != nil {
		return
	}

	if err = r.PutRef(DatasetRef{
		Peername: r.profile.Peername,
		Name:     name,
		PeerID:   r.profile.ID,
		Path:     path.String(),
	}); err != nil {
		return path, err
	}

	err = r.PutDataset(path, ds)
	return
}
