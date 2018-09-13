package actions

import (
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// ResolveDatasetRef uses a node to complete the missing pieces of a dataset
// reference. The most typical example is completing a human ref like
// peername/dataset_name with content-addressed identifiers
// It will first attempt to use the local repo to Canonicalize the reference,
// falling back to a network call if one isn't found
// TODO - this looks small now, but in the future we may consider
// reinforcing p2p network with registry lookups
func ResolveDatasetRef(n *p2p.QriNode, ref *repo.DatasetRef) (local bool, err error) {
	if err := repo.CanonicalizeDatasetRef(n.Repo, ref); err == nil && ref.Path != "" {
		return true, nil
	} else if err != nil && err != repo.ErrNotFound && err != profile.ErrNotFound {
		return false, err
	}

	return false, n.ResolveDatasetRef(ref)
}
