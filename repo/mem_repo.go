package repo

import (
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo/profile"
)

// MemRepo is an in-memory implementation of the Repo interface
type MemRepo struct {
	*MemRefstore

	store      cafs.Filestore
	filesystem qfs.Filesystem
	graph      map[string]*dsgraph.Node
	refCache   *MemRefstore
	logbook    *logbook.Book

	profile  *profile.Profile
	profiles profile.Store
}

// NewMemRepo creates a new in-memory repository
// TODO (b5) - need a better mem-repo constructor, we don't need a logbook for
// all test cases
func NewMemRepo(p *profile.Profile, store cafs.Filestore, fsys qfs.Filesystem, ps profile.Store) (*MemRepo, error) {
	book, err := logbook.NewBook(p.PrivKey, p.Peername, fsys, "/map/logbook")
	if err != nil {
		return nil, err
	}
	return &MemRepo{
		store:       store,
		filesystem:  fsys,
		MemRefstore: &MemRefstore{},
		refCache:    &MemRefstore{},
		logbook:     book,
		profile:     p,
		profiles:    ps,
	}, nil
}

// Store returns the underlying cafs.Filestore for this repo
func (r *MemRepo) Store() cafs.Filestore {
	return r.store
}

// Filesystem gives access to the underlying filesystem
func (r *MemRepo) Filesystem() qfs.Filesystem {
	return r.filesystem
}

// Logbook accesses the mem repo logbook
func (r *MemRepo) Logbook() *logbook.Book {
	return r.logbook
}

// SetFilesystem implements QFSSetter, currently used during lib contstruction
func (r *MemRepo) SetFilesystem(fs qfs.Filesystem) {
	r.filesystem = fs
}

// PrivateKey returns this repo's private key
func (r *MemRepo) PrivateKey() crypto.PrivKey {
	if r.profile == nil {
		return nil
	}
	return r.profile.PrivKey
}

// RefCache gives access to the ephemeral Refstore
func (r *MemRepo) RefCache() Refstore {
	return r.refCache
}

// Profile returns the peer profile for this repository
func (r *MemRepo) Profile() (*profile.Profile, error) {
	return r.profile, nil
}

// SetProfile updates this repo's profile
func (r *MemRepo) SetProfile(p *profile.Profile) error {
	r.profile = p
	return nil
}

// Profiles gives this repo's Peer interface implementation
func (r *MemRepo) Profiles() profile.Store {
	return r.profiles
}
