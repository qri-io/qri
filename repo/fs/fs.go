package fs_repo

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/analytics"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"io/ioutil"
	"os"
)

type Repo struct {
	basepath
	Datasets
	Namestore
	analytics Analytics
	peers     PeerStore
	cache     Datasets
}

func NewRepo(base string) (repo.Repo, error) {
	if err := os.MkdirAll(base, os.ModePerm); err != nil {
		return nil, err
	}
	bp := basepath(base)
	if err := ensureProfile(bp); err != nil {
		return nil, err
	}

	return &Repo{
		basepath:  bp,
		Datasets:  NewDatasets(base, FileDatasets),
		Namestore: Namestore{bp},
		analytics: NewAnalytics(base),
		peers:     PeerStore{bp},
		cache:     NewDatasets(base, FileCache),
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
func ensureProfile(bp basepath) error {
	if _, err := os.Stat(bp.filepath(FileProfile)); os.IsNotExist(err) {
		return bp.saveFile(&profile.Profile{}, FileProfile)
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
