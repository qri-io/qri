package fsrepo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ipfs/go-datastore"
	golog "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/doggos"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/search"
)

var log = golog.Logger("fsrepo")

// Repo is a filesystem-based implementation of the Repo interface
type Repo struct {
	pk    crypto.PrivKey
	store cafs.Filestore
	basepath
	graph map[string]*dsgraph.Node

	Datasets
	Refstore
	QueryLog
	ChangeRequests

	analytics Analytics
	peers     PeerStore
	cache     Datasets
	index     search.Index
}

// NewRepo creates a new file-based repository
func NewRepo(store cafs.Filestore, base, id string) (repo.Repo, error) {
	if err := os.MkdirAll(base, os.ModePerm); err != nil {
		return nil, err
	}
	bp := basepath(base)
	if err := ensureProfile(bp, id); err != nil {
		return nil, err
	}

	r := &Repo{
		store:    store,
		basepath: bp,

		Datasets:       NewDatasets(base, FileDatasets, store),
		Refstore:       Refstore{basepath: bp, store: store},
		QueryLog:       NewQueryLog(base, FileQueryLogs, store),
		ChangeRequests: NewChangeRequests(base, FileChangeRequests),

		analytics: NewAnalytics(base),
		peers:     PeerStore{bp},
		cache:     NewDatasets(base, FileCache, nil),
	}

	if index, err := search.LoadIndex(bp.filepath(FileSearchIndex)); err == nil {
		r.index = index
		r.Refstore.index = index
	}

	// TODO - this is racey.
	// go func() {
	// 	r.graph, _ = repo.Graph(r)
	// }()

	return r, nil
}

// Store returns the underlying cafs.Filestore driving this repo
func (r Repo) Store() cafs.Filestore {
	return r.store
}

// Graph returns the graph of dataset objects for this repo
func (r *Repo) Graph() (map[string]*dsgraph.Node, error) {
	if r.graph == nil {
		nodes, err := repo.Graph(r)
		if err != nil {
			log.Debug(err.Error())
			return nil, err
		}
		r.graph = nodes
	}
	return r.graph, nil
}

// Profile gives this repo's peer profile
func (r *Repo) Profile() (*profile.Profile, error) {
	p := &profile.Profile{}
	data, err := ioutil.ReadFile(r.filepath(FileProfile))
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		log.Debug(err.Error())
		return p, fmt.Errorf("error loading profile: %s", err.Error())
	}

	if err := json.Unmarshal(data, &p); err != nil {
		log.Debug(err.Error())
		return p, fmt.Errorf("error unmarshaling profile: %s", err.Error())
	}

	return p, nil
}

// SaveProfile updates this repo's peer profile info
func (r *Repo) SaveProfile(p *profile.Profile) error {
	return r.saveFile(p, FileProfile)
}

// ensureProfile makes sure a profile file is saved locally
// makes it easier to edit that file to change user data
func ensureProfile(bp basepath, id string) error {
	if _, err := os.Stat(bp.filepath(FileProfile)); os.IsNotExist(err) {
		return bp.saveFile(&profile.Profile{
			ID:       id,
			Peername: doggos.DoggoNick(id),
		}, FileProfile)
	}

	p := &profile.Profile{}
	data, err := ioutil.ReadFile(bp.filepath(FileProfile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		log.Debug(err.Error())
		return fmt.Errorf("error loading profile: %s", err.Error())
	}

	if err := json.Unmarshal(data, &p); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error unmarshaling profile: %s", err.Error())
	}

	if p.ID != id {
		p.ID = id
		if p.Peername == "" {
			p.Peername = doggos.DoggoNick(p.ID)
		}
		bp.saveFile(p, FileProfile)
	}

	return nil
}

// SetPrivateKey sets an internal reference to the private key for this profile
func (r *Repo) SetPrivateKey(pk crypto.PrivKey) error {
	r.pk = pk
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

// Search this repo for dataset references
func (r *Repo) Search(p repo.SearchParams) ([]repo.DatasetRef, error) {
	if r.index == nil {
		return nil, fmt.Errorf("search not supported")
	}

	refs, err := search.Search(r.index, p)
	if err != nil {
		log.Debug(err.Error())
		return refs, err
	}
	for _, ref := range refs {
		if ref.Path == "" {
			if got, err := r.GetRef(ref); err == nil {
				ref.Path = got.Path
			}
		}

		if ds, err := r.GetDataset(datastore.NewKey(ref.Path)); err == nil {
			ref.Dataset = ds
		} else {
			log.Debug(err.Error())
		}
	}
	return refs, nil
}

// UpdateSearchIndex refreshes this repos search index
func (r *Repo) UpdateSearchIndex(store cafs.Filestore) error {
	return search.IndexRepo(r, r.index)
}

// Peers returns this repo's Peers implementation
func (r *Repo) Peers() repo.Peers {
	return r.peers
}

// Cache gives this repo's ephemeral cache of datasets
func (r *Repo) Cache() repo.Datasets {
	return r.cache
}

// Analytics gets this repo's Analytics store
func (r *Repo) Analytics() analytics.Analytics {
	return r.analytics
}

// SavePeers saves a set of peers to the repo
func (r *Repo) SavePeers(p map[string]*profile.Profile) error {
	return r.saveFile(p, FilePeers)
}

// CreateDataset initializes a dataset from a dataset pointer and data file
func (r *Repo) CreateDataset(name string, ds *dataset.Dataset, data cafs.File, pin bool) (path datastore.Key, err error) {
	return dsfs.CreateDataset(r.store, ds, data, r.pk, pin)
}

// Destroy destroys this repository
func (r *Repo) Destroy() error {
	return os.RemoveAll(string(r.basepath))
}
