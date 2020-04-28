package repo

import (
	"context"
	"fmt"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event/hook"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// MemRepo is an in-memory implementation of the Repo interface
type MemRepo struct {
	*MemRefstore

	store      cafs.Filestore
	filesystem qfs.Filesystem
	graph      map[string]*dsgraph.Node
	refCache   *MemRefstore
	logbook    *logbook.Book
	dscache    *dscache.Dscache

	profile  *profile.Profile
	profiles profile.Store
}

// NewMemRepo creates a new in-memory repository
// TODO (b5) - need a better mem-repo constructor, we don't need a logbook for
// all test cases
func NewMemRepo(p *profile.Profile, store cafs.Filestore, fsys qfs.Filesystem, ps profile.Store) (*MemRepo, error) {
	book, err := logbook.NewJournal(p.PrivKey, p.Peername, fsys, "/mem/logbook")
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	// NOTE: This dscache won't get change notifications from FSI, because it's not constructed
	// with the hook for FSI.
	cache := dscache.NewDscache(ctx, fsys, []hook.ChangeNotifier{book}, p.Peername, "")

	return &MemRepo{
		store:       store,
		filesystem:  fsys,
		MemRefstore: &MemRefstore{},
		refCache:    &MemRefstore{},
		logbook:     book,
		dscache:     cache,
		profile:     p,
		profiles:    ps,
	}, nil
}

// ResolveRef implements the dsref.RefResolver interface
func (r *MemRepo) ResolveRef(ctx context.Context, ref *dsref.Ref) (string, error) {
	if r == nil {
		return "", dsref.ErrNotFound
	}

	if ref.Username == "me" {
		ref.Username = r.profile.Peername
	}

	match, err := r.GetRef(reporef.RefFromDsref(*ref))
	if err != nil {
		return "", dsref.ErrNotFound
	}

	// TODO (b5) - repo doens't store IDs yet, breaking the assertion that ResolveRef
	// will set the ID of a dataset. Need to fix that before we can ship this

	if ref.Path == "" {
		if match.FSIPath != "" {
			ref.Path = fmt.Sprintf("/fsi%s", match.FSIPath)
		} else {
			ref.Path = match.Path
		}
	}

	return "", nil
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

// Dscache returns a dscache
func (r *MemRepo) Dscache() *dscache.Dscache {
	return r.dscache
}

// RemoveLogbook drops a MemRepo's logbook pointer. MemRepo gets used in tests
// a bunch, where logbook manipulation is helpful
func (r *MemRepo) RemoveLogbook() {
	r.logbook = nil
}

// SetLogbook assigns MemRepo's logbook. MemRepo gets used in tests a bunch,
// where logbook manipulation is helpful
func (r *MemRepo) SetLogbook(book *logbook.Book) {
	r.logbook = book
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
