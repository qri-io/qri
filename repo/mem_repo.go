package repo

import (
	"github.com/qri-io/analytics"
	"github.com/qri-io/qri/repo/profile"
)

type MemRepo struct {
	MemDatasets
	MemNamestore
	profile   *profile.Profile
	peers     Peers
	cache     MemDatasets
	analytics analytics.Analytics
}

func NewRepo(p *profile.Profile, ps Peers, a analytics.Analytics) (Repo, error) {
	return &MemRepo{
		MemDatasets:  MemDatasets{},
		MemNamestore: MemNamestore{},
		profile:      p,
		peers:        ps,
		analytics:    a,
		cache:        MemDatasets{},
	}, nil
}

func (r *MemRepo) Profile() (*profile.Profile, error) {
	return r.profile, nil
}

func (r *MemRepo) SaveProfile(p *profile.Profile) error {
	r.profile = p
	return nil
}

func (r *MemRepo) Peers() Peers {
	return r.peers
}

func (r *MemRepo) Cache() Datasets {
	return r.cache
}

func (r *MemRepo) Analytics() analytics.Analytics {
	return r.analytics
}
