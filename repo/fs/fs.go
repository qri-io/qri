// Package fsrepo is a file-system implementation of repo
package fsrepo

import (
	"fmt"
	"os"

	golog "github.com/ipfs/go-log"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/registry/regclient"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

var log = golog.Logger("fsrepo")

func init() {
	golog.SetLogLevel("fsrepo", "info")
}

// Repo is a filesystem-based implementation of the Repo interface
type Repo struct {
	basepath

	repo.Refstore

	profile *profile.Profile

	store   cafs.Filestore
	fsys    qfs.Filesystem
	graph   map[string]*dsgraph.Node
	logbook *logbook.Book

	profiles *ProfileStore

	registry *regclient.Client
}

// NewRepo creates a new file-based repository
//
// Deprecated: use CreateRepo instead
func NewRepo(store cafs.Filestore, fsys qfs.Filesystem, book *logbook.Book, pro *profile.Profile, base string) (repo.Repo, error) {
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
		fsys:     fsys,
		basepath: bp,
		logbook:  book,

		Refstore: Refstore{basepath: bp, store: store, file: FileRefs},

		profiles: NewProfileStore(bp),
	}

	if _, err := maybeCreateFlatbufferRefsFile(base); err != nil {
		return nil, err
	}

	// add our own profile to the store if it doesn't already exist.
	if _, e := r.Profiles().GetProfile(pro.ID); e != nil {
		if err := r.Profiles().PutProfile(pro); err != nil {
			return nil, err
		}
	}

	return r, nil
}

// Path returns the path to the root of the repo directory
func (r Repo) Path() string {
	return string(r.basepath)
}

// Store returns the underlying cafs.Filestore driving this repo
func (r Repo) Store() cafs.Filestore {
	return r.store
}

// Filesystem returns this repo's Filesystem
func (r Repo) Filesystem() qfs.Filesystem {
	return r.fsys
}

// SetFilesystem implements QFSSetter, currently used during lib contstruction
func (r *Repo) SetFilesystem(fs qfs.Filesystem) {
	r.fsys = fs
}

// Profile gives this repo's peer profile
func (r *Repo) Profile() (*profile.Profile, error) {
	return r.profile, nil
}

// Logbook stores operation logs for coordinating state across peers
func (r *Repo) Logbook() *logbook.Book {
	return r.logbook
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

// Profiles returns this repo's Peers implementation
func (r *Repo) Profiles() profile.Store {
	return r.profiles
}

// Destroy destroys this repository
func (r *Repo) Destroy() error {
	return os.RemoveAll(string(r.basepath))
}
