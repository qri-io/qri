package fs_repo

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/cafs"
	"github.com/qri-io/doggos"
	"io/ioutil"
	"os"

	"github.com/qri-io/analytics"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/search"
)

type Repo struct {
	basepath

	Datasets
	Namestore
	QueryLog

	analytics Analytics
	peers     PeerStore
	cache     Datasets
	index     search.Index
}

func NewRepo(store cafs.Filestore, base, id string) (repo.Repo, error) {
	if err := os.MkdirAll(base, os.ModePerm); err != nil {
		return nil, err
	}
	bp := basepath(base)
	if err := ensureProfile(bp, id); err != nil {
		return nil, err
	}

	index, err := search.LoadIndex(bp.filepath(FileSearchIndex))
	if err != nil {
		return nil, err
	}

	return &Repo{
		basepath:  bp,
		Datasets:  NewDatasets(base, FileDatasets, store),
		Namestore: Namestore{bp, index, store},
		QueryLog:  NewQueryLog(base, FileQueryLogs, store),
		analytics: NewAnalytics(base),
		peers:     PeerStore{bp},
		cache:     NewDatasets(base, FileCache, nil),
		index:     index,
	}, nil
}

func (r *Repo) Profile() (*profile.Profile, error) {
	p := &profile.Profile{}
	data, err := ioutil.ReadFile(r.filepath(FileProfile))
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return p, fmt.Errorf("error loading profile: %s", err.Error())
	}

	if err := json.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("error unmarshaling profile: %s", err.Error())
	}

	return p, nil
}

func (r *Repo) SaveProfile(p *profile.Profile) error {
	return r.saveFile(p, FileProfile)
}

// ensureProfile makes sure a profile file is saved locally
// makes it easier to edit that file to change user data
func ensureProfile(bp basepath, id string) error {
	if _, err := os.Stat(bp.filepath(FileProfile)); os.IsNotExist(err) {
		return bp.saveFile(&profile.Profile{
			Id:       id,
			Username: doggos.DoggoNick(id),
		}, FileProfile)
	} else {
		p := &profile.Profile{}
		data, err := ioutil.ReadFile(bp.filepath(FileProfile))
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("error loading profile: %s", err.Error())
		}

		if err := json.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("error unmarshaling profile: %s", err.Error())
		}

		if p.Id != id {
			p.Id = id
			if p.Username == "" {
				p.Username = doggos.DoggoNick(p.Id)
			}
			bp.saveFile(p, FileProfile)
		}
	}
	return nil
}

// func (r *Repo) Peers() (map[string]*profile.Profile, error) {
// 	p := map[string]*profile.Profile{}
// 	data, err := ioutil.ReadFile(r.filepath(FilePeers))
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			return p, nil
// 		}
// 		return p, fmt.Errorf("error loading peers: %s", err.Error())
// 	}
// 	if err := json.Unmarshal(data, &p); err != nil {
// 		return p, fmt.Errorf("error unmarshaling peers: %s", err.Error())
// 	}
// 	return p, nil
// }

// fs implements the search interface
func (r *Repo) Search(p repo.SearchParams) ([]*repo.DatasetRef, error) {
	refs, err := search.Search(r.index, p)
	if err != nil {
		return refs, err
	}
	for _, ref := range refs {
		if name, err := r.GetName(ref.Path); err == nil {
			ref.Name = name
		}

		if ds, err := r.GetDataset(ref.Path); err == nil {
			ref.Dataset = ds
		} else {
			// fmt.Println(err.Error())
		}
	}
	return refs, nil
}

func (r *Repo) UpdateSearchIndex(store cafs.Filestore) error {
	return search.IndexRepo(store, r, r.index)
}

func (r *Repo) Peers() repo.Peers {
	return r.peers
}

func (r *Repo) Cache() repo.Datasets {
	return r.cache
}

func (r *Repo) Analytics() analytics.Analytics {
	return r.analytics
}

func (r *Repo) SavePeers(p map[string]*profile.Profile) error {
	return r.saveFile(p, FilePeers)
}

func (r *Repo) Destroy() error {
	return os.RemoveAll(string(r.basepath))
}
