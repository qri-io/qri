// Repo represents a repository of qri information
// Analogous to a git repository, repo expects a
// specific structure.
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
	ErrNotFound = fmt.Errorf("repo: not found")
)

// Repo is the interface for working with a qri repository
// conceptually, it's a more-specific version of a datastore.
type Repo interface {
	// The primary purpose of a qri repo is to serve as
	// a store of datasets. The behaviour of the embedded
	// DatasetStore will typically differ from the
	Datasets
	//
	Namestore
	// A repository must maintain profile information
	// about the owner of this dataset
	Profile() (*profile.Profile, error)
	// It must be possible to save
	SaveProfile(*profile.Profile) error
	// A repository must maintain profile information
	// about encountered peers. Decsisions regarding
	// retentaion of peers is left to the the implementation
	Peers() Peers
	// Cache keeps an ephemeral store of dataset information
	// that may be purged at any moment
	Cache() Datasets
	// All repositories provide their own analytics information
	Analytics() analytics.Analytics
}

// Dataset store is the minimum interface to act as a
// store of datasets
type Datasets interface {
	// Query is extracted from the ipfs datastore interface:
	// github.com/ipfs/go-datastore
	// We track with the query package to support a uniform
	// querying interface between datastores & these custom
	// stores
	Query(query.Query) (query.Results, error)
	PutDataset(path datastore.Key, ds *dataset.Dataset) error
	PutDatasets([]*dataset.DatasetRef) error
	GetDataset(path datastore.Key) (*dataset.Dataset, error)
	DeleteDataset(path datastore.Key) error
}

func DatasetsQuery(dss Datasets, q query.Query) (map[string]*dataset.Dataset, error) {
	// i := 0
	ds := map[string]*dataset.Dataset{}
	results, err := dss.Query(q)
	if err != nil {
		return nil, err
	}

	// if q.Limit != 0 {
	// 	ds = make([]*dataset.Dataset, q.Limit)
	// }

	for res := range results.Next() {
		d, ok := res.Value.(*dataset.Dataset)
		if !ok {
			return nil, fmt.Errorf("query returned the wrong type, expected a profile pointer")
		}
		ds[res.Key] = d
		// if q.Limit != 0 {
		// 	ds[i] = d
		// } else {
		// 	ds = append(ds, d)
		// }
		// i++
	}

	// if q.Limit != 0 {
	// 	ds = ds[:i]
	// }

	return ds, nil
}
