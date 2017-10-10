// Repo represents a repository of qri information
// Analogous to a git repository, repo expects a rigid structure
// filled with rich types specific to qri.
// Lots of things in here take inspiration from the ipfs datastore interface:
// github.com/ipfs/go-datastore
package repo

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/analytics"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo/profile"
)

var (
	// implementers should return this variable when stuff isn't found
	ErrNotFound = fmt.Errorf("repo: not found")
)

// Repo is the interface for working with a qri repository
// conceptually, it's a more-specific version of a datastore.
type Repo interface {
	// At the heart of all repositories is a namestore, which maps user-defined
	// aliases for datasets to their underlying content-addressed hash
	// as an example:
	// 		my_dataset : /ipfs/Qmeiuzejjs....
	// these aliases are then used in qri SQL statements. Names are *not*
	// universally unique, but must be unique within the namestore
	Namestore
	// Repos also serve as a store of dataset information.
	// It's important that this store maintain sync with any underlying filestore.
	// (which is why we might want to kill this in favor of just having a cache?)
	// The behaviour of the embedded DatasetStore will typically differ from the cache,
	// by only returning saved/pinned/permanent datasets.
	Datasets
	// A repository must maintain profile information about the owner of this dataset.
	// The value returned by Profile() should represent the user.
	Profile() (*profile.Profile, error)
	// It must be possible to alter profile information.
	SaveProfile(*profile.Profile) error
	// A repository must maintain profile information about encountered peers.
	// Decsisions regarding retentaion of peers is left to the the implementation
	Peers() Peers
	// Cache keeps an ephemeral store of dataset information
	// that may be purged at any moment. Results of searching for datasets,
	// dataset references other users have, etc, should all be stored here.
	Cache() Datasets
	// All repositories provide their own analytics information.
	// Our analytics implementation is under super-active development.
	Analytics() analytics.Analytics
}

// Dataset store is the minimum interface to act as a store of datasets.
// It's intended to look a *lot* like the ipfs datastore interface, but
// scoped only to datasets to make for easier consumption.
// Datasets stored here should be reasonably dereferenced to avoid
// additional lookups.
// All fields here work only with paths (which are datastore.Key's)
// to dereference a name, you'll need a Namestore interface
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

// DatasetRef encapsulates a reference to a dataset. This needs to exist to bind
// ways of referring to a dataset to a dataset itself, as datasets can't easily
// contain their own hash information, and names are unique on a per-repository
// basis.
// It's tempting to think this needs to be "bigger", supporting more fields,
// keep in mind that if the information is important at all, it should
// be stored as metadata within the dataset itself.
type DatasetRef struct {
	// The dataset being referenced
	Dataset *dataset.Dataset `json:"dataset"`
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path datastore.Key `json:"path"`
}

// Searchable is an opt-in interface for supporting repository search
type Searchable interface {
	Search(q string) (string, error)
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
