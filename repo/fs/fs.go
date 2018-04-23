package fsrepo

import (
	"fmt"
	"os"

	golog "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/actions"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/search"
)

var log = golog.Logger("fsrepo")

func init() {
	golog.SetLogLevel("fsrepo", "info")
}

// Repo is a filesystem-based implementation of the Repo interface
type Repo struct {
	basepath

	profile *profile.Profile
	pk      crypto.PrivKey

	store cafs.Filestore
	graph map[string]*dsgraph.Node

	Refstore
	EventLog

	profiles ProfileStore
	index    search.Index
}

// NewRepo creates a new file-based repository
func NewRepo(store cafs.Filestore, pro *profile.Profile, base string) (repo.Repo, error) {
	if err := os.MkdirAll(base, os.ModePerm); err != nil {
		return nil, err
	}
	bp := basepath(base)

	if pro.PrivKey == nil {
		return nil, fmt.Errorf("Expected: PrivateKey")
	}

	r := &Repo{
		profile: pro,
		pk:      pro.PrivKey,

		store:    store,
		basepath: bp,

		Refstore: Refstore{basepath: bp, store: store, file: FileRefstore},
		EventLog: NewEventLog(base, FileEventLogs, store),

		profiles: NewProfileStore(bp),
	}

	if index, err := search.LoadIndex(bp.filepath(FileSearchIndex)); err == nil {
		r.index = index
		r.Refstore.index = index
	}

	// TODO - this is racey.
	// go func() {
	// 	r.graph, _ = repo.Graph(r)
	// }()

	if err := r.Profiles().PutProfile(pro); err != nil {
		return nil, err
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

// SetPrivateKey sets an internal reference to the private key for this profile
func (r *Repo) SetPrivateKey(pk crypto.PrivKey) error {
	r.pk = pk
	return nil
}

// PrivateKey returns this repo's private key
func (r *Repo) PrivateKey() crypto.PrivKey {
	return r.pk
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

		act := actions.Dataset{r}
		if err := act.ReadDataset(&ref); err != nil {
			log.Debug(err.Error())
		}
	}
	return refs, nil
}

// UpdateSearchIndex refreshes this repos search index
func (r *Repo) UpdateSearchIndex(store cafs.Filestore) error {
	return search.IndexRepo(r, r.index)
}

// Profiles returns this repo's Peers implementation
func (r *Repo) Profiles() profile.Store {
	return r.profiles
}

// Destroy destroys this repository
func (r *Repo) Destroy() error {
	return os.RemoveAll(string(r.basepath))
}
