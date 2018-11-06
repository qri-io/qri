package fsrepo

import (
	"encoding/json"
	"fmt"
	"os"

	golog "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/search"
	"github.com/qri-io/registry/regclient"
)

var log = golog.Logger("fsrepo")

func init() {
	golog.SetLogLevel("fsrepo", "info")
}

// Repo is a filesystem-based implementation of the Repo interface
type Repo struct {
	basepath

	Refstore
	EventLog

	profile *profile.Profile

	store        cafs.Filestore
	selectedRefs []repo.DatasetRef
	graph        map[string]*dsgraph.Node

	profiles *ProfileStore
	index    search.Index

	registry *regclient.Client
}

// NewRepo creates a new file-based repository
func NewRepo(store cafs.Filestore, pro *profile.Profile, rc *regclient.Client, base string) (repo.Repo, error) {
	if err := os.MkdirAll(base, os.ModePerm); err != nil {
		return nil, err
	}
	bp := basepath(base)

	if pro.PrivKey == nil {
		return nil, fmt.Errorf("Expected: PrivateKey")
	}

	r := &Repo{
		profile: pro,

		store:    store,
		basepath: bp,

		Refstore: Refstore{basepath: bp, store: store, file: FileRefstore},
		EventLog: NewEventLog(base, FileEventLogs, store),

		profiles: NewProfileStore(bp),

		registry: rc,
	}

	if index, err := search.LoadIndex(bp.filepath(FileSearchIndex)); err == nil {
		r.index = index
		r.Refstore.index = index
	}

	// add our own profile to the store if it doesn't already exist.
	if _, e := r.Profiles().GetProfile(pro.ID); e != nil {
		if err := r.Profiles().PutProfile(pro); err != nil {
			return nil, err
		}
	}

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
	return r.profile, nil
}

// SetProfile updates this repo's peer profile info
func (r *Repo) SetProfile(p *profile.Profile) error {
	r.profile = p
	return r.Profiles().PutProfile(p)
}

// PrivateKey returns this repo's private key
func (r *Repo) PrivateKey() crypto.PrivKey {
	return r.profile.PrivKey
}

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

		if err := base.ReadDataset(r, &ref); err != nil {
			log.Debug(err.Error())
		}
	}
	return refs, nil
}

// UpdateSearchIndex refreshes this repos search index
func (r *Repo) UpdateSearchIndex(store cafs.Filestore) error {
	return search.IndexRepo(r, r.index)
}

// SetSelectedRefs sets the current reference selection
func (r *Repo) SetSelectedRefs(sel []repo.DatasetRef) error {
	return r.saveFile(sel, FileSelectedRefs)
}

// SelectedRefs gives the current reference selection
func (r *Repo) SelectedRefs() ([]repo.DatasetRef, error) {
	data, err := r.readBytes(FileSelectedRefs)
	if err != nil {
		return nil, nil
	}
	res := []repo.DatasetRef{}
	if err = json.Unmarshal(data, &res); err != nil {
		return nil, nil
	}

	return res, nil
}

// Profiles returns this repo's Peers implementation
func (r *Repo) Profiles() profile.Store {
	return r.profiles
}

// Registry returns a client for interacting with a federated registry if one exists, otherwise nil
func (r *Repo) Registry() *regclient.Client {
	return r.registry
}

// Destroy destroys this repository
func (r *Repo) Destroy() error {
	return os.RemoveAll(string(r.basepath))
}
