package mem_repo

import (
	"encoding/json"
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
	"io/ioutil"
	"os"
	"path/filepath"
)

type Repo struct {
	base      string
	profile   *profile.Profile
	analytics analytics.Analytics
	peers     peers.Peers
	cache     *Repo
}

// func NewRepo(base string) (repo.Repo, error) {
// 	if err := os.MkdirAll(base, os.ModePerm); err != nil {
// 		return nil, err
// 	}
// 	return &Repo{
// 		base: base,
// 	}, nil
// }

func (r *Repo) AddDataset(path string, ds *dataset.Dataset) error {

}

func (r *Repo) DeleteDataset(path string) error {

}

func (r *Repo) Query(query.Query) (query.Results, error) {

}

func (r *Repo) Profile() (*profile.Profile, error) {
	return r.profile, nil
}

func (r *Repo) SaveProfile(p *profile.Profile) error {
	r.profile = p
	return nil
}
