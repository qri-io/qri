// Package repo represents a repository of qri information
// Analogous to a git repository, repo expects a rigid structure
// filled with rich types specific to qri.
// Lots of things in here take inspiration from the ipfs datastore interface:
// github.com/ipfs/go-datastore
package repo

import (
	"fmt"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/analytics"
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
	Graph() (map[string]*dsgraph.Node, error)
	// All Repos must keep a Refstore, defining a given peer's datasets
	Refstore
	// CreateDataset initializes a dataset from a dataset pointer and data file
	// It's not part of the Datasets interface because creating a dataset requires
	// access to this repos store & private key
	CreateDataset(name string, ds *dataset.Dataset, data cafs.File, pin bool) (path datastore.Key, err error)
	// Repos also serve as a store of dataset information.
	// It's important that this store maintain sync with any underlying filestore.
	// (which is why we might want to kill this in favor of just having a cache?)
	// The behaviour of the embedded DatasetStore will typically differ from the cache,
	// by only returning saved/pinned/permanent datasets.
	Datasets
	// QueryLog keeps a log of queries that have been run
	QueryLog
	// ChangeRequets gives this repo's change request store
	ChangeRequestStore
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
	Peers() Peers
	// Cache keeps an ephemeral store of dataset information
	// that may be purged at any moment. Results of searching for datasets,
	// dataset references other peers have, etc, should all be stored here.
	Cache() Datasets
	// All repositories provide their own analytics information.
	// Our analytics implementation is under super-active development.
	Analytics() analytics.Analytics
}

// Datasets is the minimum interface to act as a store of datasets.
// It's intended to look a *lot* like the ipfs datastore interface, but
// scoped only to datasets to make for easier consumption.
// Datasets stored here should be reasonably dereferenced to avoid
// additional lookups.
// All fields here work only with paths (which are datastore.Key's)
// to dereference a name, you'll need a Refstore interface
// oh golang, can we haz generics plz?
type Datasets interface {
	// Put a dataset in the store
	PutDataset(path datastore.Key, ds *dataset.Dataset) error
	// Put multiple datasets in the store
	PutDatasets([]*DatasetRef) error
	// Get a dataset from the store
	GetDataset(path datastore.Key) (*dataset.Dataset, error)
	// Remove a dataset from the store
	DeleteDataset(path datastore.Key) error
	// Query is extracted from the ipfs datastore interface:
	Query(query.Query) (query.Results, error)
}

// QueryLogItem is a list of details for logging a query
type QueryLogItem struct {
	Query       string
	Name        string
	Key         datastore.Key
	DatasetPath datastore.Key
	Time        time.Time
}

// QueryLog keeps logs
type QueryLog interface {
	LogQuery(*QueryLogItem) error
	ListQueryLogs(limit, offset int) ([]*QueryLogItem, error)
	QueryLogItem(q *QueryLogItem) (*QueryLogItem, error)
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

// DatasetsQuery is a convenience function to read all query results & parse into a
// map[string]*dataset.Dataset.
func DatasetsQuery(dss Datasets, q query.Query) (map[string]*dataset.Dataset, error) {
	ds := map[string]*dataset.Dataset{}
	results, err := dss.Query(q)
	if err != nil {
		return nil, err
	}

	for res := range results.Next() {
		d, ok := res.Value.(*dataset.Dataset)
		if !ok {
			return nil, fmt.Errorf("query returned the wrong type, expected a profile pointer")
		}
		ds[res.Key] = d
	}

	return ds, nil
}
