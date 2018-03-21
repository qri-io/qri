// Package repo represents a repository of qri information
// Analogous to a git repository, repo expects a rigid structure
// filled with rich types specific to qri.
// Lots of things in here take inspiration from the ipfs datastore interface:
// github.com/ipfs/go-datastore
package repo

import (
	"fmt"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo/profile"
)

var (
	// ErrNotFound is the err implementers should return when stuff isn't found
	ErrNotFound = fmt.Errorf("repo: not found")
	// ErrPeerIDRequired is for when a peerID is missing-but-expected
	ErrPeerIDRequired = fmt.Errorf("repo: peerID is required")
	// ErrPeernameRequired is for when a peername is missing-but-expected
	ErrPeernameRequired = fmt.Errorf("repo: peername is required")
	// ErrNameRequired is for when a name is missing-but-expected
	ErrNameRequired = fmt.Errorf("repo: name is required")
	// ErrPathRequired is for when a path is missing-but-expected
	ErrPathRequired = fmt.Errorf("repo: path is required")
	// ErrNameTaken is for when a name name is already taken
	ErrNameTaken = fmt.Errorf("repo: name already in use")
	// ErrRepoEmpty is for when the repo has no datasets
	ErrRepoEmpty = fmt.Errorf("repo: this repo contains no datasets")
	// ErrNotPinner is for when the repo doesn't have the concept of pinning as a feature
	ErrNotPinner = fmt.Errorf("repo: backing store doesn't support pinning")
)

// Repo is the interface for working with a qri repository qri repos are stored
// graph of resources:datasets, known peers, analytics data, change requests, etc.
// Repos are connected to a single peer profile.
// Repos must wrap an underlying cafs.Filestore, which
// is intended to act as the canonical store of state across all peers
// that this repo may interact with.
type Repo interface {
	// All repositories wraps a content-addressed filestore as the cannonical
	// record of this repository's data. Store gives direct access to the
	// cafs.Filestore instance any given repo is using.
	Store() cafs.Filestore

	// Graph returns a graph of this repositories data resources
	// TODO - either depricate or refactor this. We need graph support for dataset transforms
	// to work properly, but the whole notion of transforms needs a rethink first.
	Graph() (map[string]*dsgraph.Node, error)

	// All Repos must keep a Refstore, defining a given peer's datasets
	Refstore
	// Caches of dataset references
	RefCache() Refstore

	// Repos also serve as a store of dataset information.
	// It's important that this store maintain sync with any underlying filestore.
	// (which is why we might want to kill this in favor of just having a cache?)
	// The behaviour of the embedded DatasetStore will typically differ from the cache,
	// by only returning saved/pinned/permanent datasets.
	Datasets

	// EventLog keeps a log of Profile activity for this repo
	EventLog

	// A repository must maintain profile information about the owner of this dataset.
	// The value returned by Profile() should represent the peer.
	Profile() (*profile.Profile, error)
	// It must be possible to alter profile information.
	SaveProfile(*profile.Profile) error
	// SetPrivateKey sets an internal reference to the private key for this profile.
	// PrivateKey is used to tie peer actions to this profile. Repo implementations must
	// never expose this private key once set.
	SetPrivateKey(pk crypto.PrivKey) error
	// A repository must maintain profile information about encountered peers.
	// Decsisions regarding retentaion of peers is left to the the implementation
	// TODO - should rename this to "profiles" to separate from the networking
	// concept of a peer
	Profiles() Profiles
}

// Datasets is the minimum interface to act as a store of datasets.
// Datasets stored here should be reasonably dereferenced to avoid
// additional lookups.
// All fields here work only with paths (which are datastore.Key's)
type Datasets interface {
	// CreateDataset initializes a dataset from a dataset pointer and data file
	// It's not part of the Datasets interface because creating a dataset requires
	// access to this repos store & private key
	CreateDataset(name string, ds *dataset.Dataset, data cafs.File, pin bool) (ref DatasetRef, err error)
	ReadDataset(ref *DatasetRef) error
	RenameDataset(a, b DatasetRef) error
	DeleteDataset(ref DatasetRef) (err error)
	PinDataset(ref DatasetRef) error
	UnpinDataset(ref DatasetRef) error
}

// SearchParams encapsulates parameters provided to Searchable.Search
type SearchParams struct {
	Q             string
	Limit, Offset int
}

// Searchable is an opt-in interface for supporting repository search
type Searchable interface {
	Search(p SearchParams) ([]DatasetRef, error)
}
