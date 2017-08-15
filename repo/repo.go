// Repo represents a repository of qri information
// Analogous to a git repository, repo expects a
// specific structure.
package repo

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsgraph"
)

// Repo is the uniform interface
type Repo interface {
	Namespace() (map[string]datastore.Key, error)
	SaveNamespace(map[string]datastore.Key) error

	QueryResults() (dsgraph.QueryResults, error)
	SaveQueryResults(dsgraph.QueryResults) error

	ResourceMeta() (dsgraph.ResourceMeta, error)
	SaveResourceMeta(dsgraph.ResourceMeta) error

	ResourceQueries() (dsgraph.ResourceQueries, error)
	SaveResourceQueries(dsgraph.ResourceQueries) error

	Destroy() error
}
