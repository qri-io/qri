// Repo represents a repository of qri information
// Analogous to a git repository, repo expects a
// specific structure.
package repo

import (
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/dataset"
	// "github.com/qri-io/dataset"
	// "github.com/qri-io/dataset/dsgraph"
	// "github.com/ipfs/go-datastore"
	// "github.com/qri-io/qri/repo/peer_repo"
	"github.com/qri-io/analytics"
	"github.com/qri-io/qri/repo/peers"
	"github.com/qri-io/qri/repo/profile"
)

// Repo is the uniform interface of a qri repository
type Repo interface {
	// The primary purpose of a qri repo is to serve as
	// a store of datasets. The behaviour of the embedded
	// DatasetStore will typically differ from the
	DatasetStore
	// A repository must maintain profile information
	// about the owner of this dataset
	Profile() (*profile.Profile, error)
	// It must be possible to save
	SaveProfile(*profile.Profile) error
	// A repository must maintain profile information
	// about encountered peers. Decsisions regarding
	// retentaion of peers is left to the the implementation
	Peers() peers.Peers
	// Cache keeps an ephemeral store of dataset information
	// that may be purged at any moment
	Cache() DatasetStore
	// All repositories provide their own analytics information
	Analytics() analytics.Analytics
}

// Dataset store is the minimum interface to act as a
// store of datasets
type DatasetStore interface {
	// Query is extracted from the ipfs datastore interface:
	// github.com/ipfs/go-datastore
	// We track with the query package to support a uniform
	// querying interface between datastores & these custom
	// stores
	Query(query.Query) (query.Results, error)
	PutDataset(path string, ds *dataset.Dataset) error
	GetDataset(path string) (*dataset.Dataset, error)
	DeleteDataset(path string) error
}
