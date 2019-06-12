package actions

import (
	"fmt"

	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry"
)

// ResolveDatasetRef uses a node to complete the missing pieces of a dataset
// reference. The most typical example is completing a human ref like
// peername/dataset_name with content-addressed identifiers
// It will first attempt to use the local repo to Canonicalize the reference,
// falling back to a network call if one isn't found
// TODO - this looks small now, but in the future we may consider
// reinforcing p2p network with registry lookups
// TODO (ramfox) - Canonicalizing a Dataset with no errors is not a good enough tell to see
// if a dataset is local or not, we have to actually attempt to load it.
// however, if we are connected to a network, we cannot fully reason if a file
// is local or from the network. We need to build tools that allow us better
// control over local only and network actions. Once we have those, we can attempt
// to load the dataset locally, if it error with DatasetNotFound, or something similar
// we will know that the dataset does not exist locally
func ResolveDatasetRef(node *p2p.QriNode, ref *repo.DatasetRef) (local bool, err error) {
	if err := repo.CanonicalizeDatasetRef(node.Repo, ref); err == nil && ref.Path != "" {
		return true, nil
	} else if err != nil && err != repo.ErrNotFound && err != profile.ErrNotFound {
		// return early on any non "not found" error
		return false, err
	}

	type response struct {
		Ref   *repo.DatasetRef
		Error error
	}

	responses := make(chan response)
	tasks := 0

	if rc := node.Repo.Registry(); rc != nil {
		tasks++

		refCopy := &repo.DatasetRef{
			Peername:  ref.Peername,
			Name:      ref.Name,
			ProfileID: ref.ProfileID,
			Path:      ref.Path,
		}

		go func(ref *repo.DatasetRef) {
			res := response{Ref: ref}
			defer func() {
				responses <- res
			}()

			var rds *registry.Dataset
			if rds, res.Error = rc.GetDataset(ref.Peername, ref.Name, ref.ProfileID.String(), ref.Path); res.Error == nil {
				// Commit author is required to resolve ref
				if rds.Commit != nil && rds.Commit.Author != nil {
					ref.Peername = rds.Handle
					ref.Name = rds.Name
					ref.ProfileID, _ = profile.IDB58Decode(rds.Commit.Author.ID)
					ref.Path = rds.Path
					return
				}
			}
		}(refCopy)
	}

	if node.Online {
		tasks++
		go func() {
			err := node.ResolveDatasetRef(ref)
			log.Debugf("p2p ref res: %s", ref)
			if !ref.Complete() && err == nil {
				err = fmt.Errorf("p2p network responded with incomplete reference")
			}
			responses <- response{Ref: ref, Error: err}
		}()
	}

	if tasks == 0 {
		return false, fmt.Errorf("node is not online and no registry is configured")
	}

	success := false
	for i := 0; i < tasks; i++ {
		res := <-responses
		err = res.Error
		if err == nil {
			success = true
			*ref = *res.Ref
			break
		}
	}

	if !success {
		return false, fmt.Errorf("error resolving ref: %s", err)
	}
	return false, nil
}
