package actions

import (
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// ResolveDatasetRef uses a node to complete the missing pieces of a dataset
// reference. The most typical example is completing a human ref like
// peername/dataset_name with content-addressed identifiers
// It will first attempt to use the local repo to Canonicalize the reference,
// falling back to a network call if one isn't found
// TODO - consider reinforcing p2p network with registry lookups
func ResolveDatasetRef(n *p2p.QriNode, ref *repo.DatasetRef) error {
	if err := repo.CanonicalizeDatasetRef(n.Repo, ref); err == nil && ref.Path != "" {
		return nil
	}

	return n.ResolveDatasetRef(ref)
}
