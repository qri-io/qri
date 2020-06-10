// Package fsrepo is a file-system implementation of repo
package fsrepo

import (
	"context"
	"fmt"
	"os"

	golog "github.com/ipfs/go-log"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
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
	dscache *dscache.Dscache

	profiles *ProfileStore
}

// NewRepo creates a new file-based repository
func NewRepo(store cafs.Filestore, fsys qfs.Filesystem, book *logbook.Book, cache *dscache.Dscache, pro *profile.Profile, base string) (repo.Repo, error) {
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
		dscache:  cache,

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

// ResolveRef implements the dsref.RefResolver interface
func (r *Repo) ResolveRef(ctx context.Context, ref *dsref.Ref) (string, error) {
	if r == nil {
		return "", dsref.ErrRefNotFound
	}

	// TODO (b5) - not totally sure why, but memRepo doesn't seem to be wiring up
	// dscache correctly in in tests
	// if r.dscache != nil {
	// 	return r.dscache.ResolveRef(ctx, ref)
	// }
	// See: https://github.com/qri-io/qri/issues/1385

	// Lookup using refstore
	datasetRef := reporef.RefFromDsref(*ref)
	foundRef, err := r.GetRef(datasetRef)
	if err != nil {
		return "", dsref.ErrRefNotFound
	}

	ref.Username = foundRef.Peername
	// ResolveRef spec expects empty ProfileID in resulting ref
	ref.ProfileID = ""
	ref.Name = foundRef.Name
	if ref.Path == "" {
		ref.Path = foundRef.Path
	}

	// Try to add initID using logbook. Ignore errors; while we should move to a world where
	// initID is used everywhere, some subsystems don't need InitID, so we shouldn't break
	// functionality.
	if r.logbook != nil {
		initID, err := r.logbook.RefToInitID(*ref)
		if err == nil {
			ref.InitID = initID
		} else {
			err = nil
		}
	}

	return "", nil
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

// Dscache returns a dscache
func (r *Repo) Dscache() *dscache.Dscache {
	return r.dscache
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
