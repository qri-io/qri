// Repo represents a repository of qri information
// Analogous to a git repository, repo expects a
// specific structure.
package repo

import (
	"github.com/ipfs/go-datastore"
	// "github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo/peer_repo"
	"github.com/qri-io/qri/repo/profile"
)

// Repo is the uniform interface
type Repo interface {
	Profile() (*profile.Profile, error)
	SaveProfile(*profile.Profile) error

	Namespace() (map[string]datastore.Key, error)
	SaveNamespace(map[string]datastore.Key) error

	// Datasets() ([]*dataset.Dataset, error)

	QueryResults() (dsgraph.QueryResults, error)
	SaveQueryResults(dsgraph.QueryResults) error

	ResourceMeta() (dsgraph.ResourceMeta, error)
	SaveResourceMeta(dsgraph.ResourceMeta) error

	ResourceQueries() (dsgraph.ResourceQueries, error)
	SaveResourceQueries(dsgraph.ResourceQueries) error

	Peers() (map[string]*peer_repo.Repo, error)
	SavePeers(map[string]*peer_repo.Repo) error

	Destroy() error
}
