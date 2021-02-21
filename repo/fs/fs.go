// Package fsrepo is a file-system implementation of repo
package fsrepo

import (
	"context"
	"fmt"
	"os"
	"sync"

	golog "github.com/ipfs/go-log"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

var log = golog.Logger("fsrepo")

// Repo is a filesystem-based implementation of the Repo interface
type Repo struct {
	basepath

	repo.Refstore

	bus     event.Bus
	fsys    *muxfs.Mux
	logbook *logbook.Book
	dscache *dscache.Dscache

	profiles profile.Store

	doneWg  sync.WaitGroup
	doneCh  chan struct{}
	doneErr error
}

var _ repo.Repo = (*Repo)(nil)

// NewRepo creates a new file-based repository
func NewRepo(path string, fsys *muxfs.Mux, book *logbook.Book, cache *dscache.Dscache, pro profile.Store, bus event.Bus) (repo.Repo, error) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Error(err)
		return nil, err
	}
	bp := basepath(path)

	r := &Repo{
		bus:      bus,
		fsys:     fsys,
		basepath: bp,
		logbook:  book,
		dscache:  cache,

		Refstore: Refstore{basepath: bp, file: FileRefs},
		profiles: pro,

		doneCh: make(chan struct{}),
	}

	r.doneWg.Add(1)
	go func() {
		<-r.fsys.Done()
		r.doneErr = r.fsys.DoneErr()
		r.doneWg.Done()
	}()

	go func() {
		r.doneWg.Wait()
		close(r.doneCh)
	}()

	if _, err := maybeCreateFlatbufferRefsFile(path); err != nil {
		return nil, err
	}

	own := pro.Owner()
	// add our own profile to the store if it doesn't already exist.
	if _, e := r.Profiles().GetProfile(own.ID); e != nil {
		if err := r.Profiles().PutProfile(own); err != nil {
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

	if r.logbook == nil {
		return "", fmt.Errorf("cannot resolve local references without logbook")
	}

	// Preserve the input ref path, and convert to the old style dataset ref for repo.
	origPath := ref.Path
	datasetRef := reporef.DatasetRef{
		Peername: ref.Username,
		Name:     ref.Name,
	}

	// Get the reference from the refstore. This has everything but initID
	match, err := r.GetRef(datasetRef)
	if err != nil {
		return "", dsref.ErrRefNotFound
	}
	// Create our resolved reference. If the input ref had a path, reassign that
	*ref = reporef.ConvertToDsref(match)
	if origPath != "" {
		ref.Path = origPath
	}

	// Get just the initID from logbook
	ref.InitID, err = r.logbook.RefToInitID(*ref)
	return "", err
}

// Path returns the path to the root of the repo directory
func (r *Repo) Path() string {
	return string(r.basepath)
}

// Bus accesses the repo's bus
func (r *Repo) Bus() event.Bus {
	return r.bus
}

// Filesystem returns this repo's Filesystem
func (r *Repo) Filesystem() *muxfs.Mux {
	return r.fsys
}

// SetFilesystem implements QFSSetter, currently used during lib contstruction
func (r *Repo) SetFilesystem(fs *muxfs.Mux) {
	r.fsys = fs
}

// Profile gives this repo's peer profile
func (r *Repo) Profile(ctx context.Context) (*profile.Profile, error) {
	return r.profiles.Active(ctx), nil
}

// Logbook stores operation logs for coordinating state across peers
func (r *Repo) Logbook() *logbook.Book {
	return r.logbook
}

// Dscache returns a dscache
func (r *Repo) Dscache() *dscache.Dscache {
	return r.dscache
}

// PrivateKey returns this repo's private key
func (r *Repo) PrivateKey(ctx context.Context) crypto.PrivKey {
	return r.profiles.Active(ctx).PrivKey
}

// Profiles returns this repo's Peers implementation
func (r *Repo) Profiles() profile.Store {
	return r.profiles
}

// Done returns a channel that the repo will send on when the repo is finished
// closing
func (r *Repo) Done() <-chan struct{} {
	return r.doneCh
}

// DoneErr gives an error that occurred during the shutdown process
func (r *Repo) DoneErr() error {
	return r.doneErr
}

// Destroy destroys this repository
func (r *Repo) Destroy() error {
	return os.RemoveAll(string(r.basepath))
}
