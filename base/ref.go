package base

import (
	"fmt"

	"github.com/qri-io/qri/repo"
)

// InLocalNamespace checks if a dataset ref is local, assumes the reference is already canonicalized
func InLocalNamespace(r repo.Repo, ref *repo.DatasetRef) bool {
	p, err := r.Profile()
	if err != nil {
		return false
	}

	return p.ID == ref.ProfileID
}

// SetPublishStatus updates the Published field of a dataset ref
func SetPublishStatus(r repo.Repo, ref *repo.DatasetRef) error {
	if err := repo.CanonicalizeDatasetRef(r, ref); err != nil {
		return err
	}

	if !InLocalNamespace(r, ref) {
		return fmt.Errorf("can't publish datsets that are not in your namespace")
	}

	return r.PutRef(*ref)
}
