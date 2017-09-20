// mem_repo is an in-memory implementation of
// the Repo interface
package mem_repo

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/analytics"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/peer_repo"
	"github.com/qri-io/qri/repo/peers"
	"github.com/qri-io/qri/repo/profile"
)

type Repo struct {
	datasets  Datasets
	profile   *profile.Profile
	peers     peers.Peers
	cache     *Repo
	analytics analytics.Analytics
}

func NewRepo(p *profile.Profile, ps peers.Peers, a analytics.Analytics) (repo.Repo, error) {
	return &Repo{
		datasets:  map[string]*dataset.Dataset{},
		profile:   p,
		peers:     ps,
		analytics: a,
		cache:     Datasets{},
	}, nil
}

func (r *Repo) Profile() (*profile.Profile, error) {
	return r.profile, nil
}

func (r *Repo) SaveProfile(p *profile.Profile) error {
	r.profile = p
	return nil
}

func (r *Repo) Peers() peers.Peers {
	return r.peers
}

func (r *Repo) Cache() repo.DatasetStore {
	return r.cache
}

func (r *Repo) Analytics() analytics.Analytics {
	return r.analytics
}
