package actions

import (
	"fmt"

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
func ResolveDatasetRef(node *p2p.QriNode, ref *repo.DatasetRef) (local bool, err error) {
	if err := repo.CanonicalizeDatasetRef(node.Repo, ref); err == nil && ref.Path != "" {
		return true, nil
	} else if err != nil && err != repo.ErrNotFound && err != profile.ErrNotFound {
		// return early on any non "not found" error
		return false, err
	}

	errs := make(chan error)
	tasks := 0

	// default error, will be cleared below while reading from errs channel if either task is assigned
	err = fmt.Errorf("node is not online and no registry is configured")

	if rc := node.Repo.Registry(); rc != nil {
		tasks++
		go func() {
			if ds, err := rc.GetDataset(ref.Peername, ref.Name, ref.ProfileID.String(), ref.Path); err == nil {
				// Commit author is required to resolve ref
				if ds.Commit != nil && ds.Commit.Author != nil {
					ref.Peername = ds.Peername
					ref.Name = ds.Name
					ref.ProfileID, _ = profile.IDB58Decode(ds.Commit.Author.ID)
					ref.Path = ds.Path
					errs <- nil
					return
				}
			}
			errs <- fmt.Errorf("not found")
		}()
	}

	if node.Online {
		tasks++
		go func() {
			errs <- node.ResolveDatasetRef(ref)
		}()
	}

	success := false
	for i := 0; i < tasks; i++ {
		if err = <-errs; err == nil {
			success = true
			break
		}
	}

	if !success {
		return false, fmt.Errorf("error resolving ref: %s", err)
	}
	return false, nil
}
