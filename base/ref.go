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
func SetPublishStatus(r repo.Repo, ref *repo.DatasetRef, published bool) error {
	if !InLocalNamespace(r, ref) {
		return fmt.Errorf("can't publish datasets that are not in your namespace")
	}

	ref.Published = published
	return r.PutRef(*ref)
}

// ToDatasetRef parses the dataset ref and returns it, allowing datasets with no history only
// if FSI is enabled.
func ToDatasetRef(path string, r repo.Repo, allowFSI bool) (*repo.DatasetRef, error) {
	if path == "" {
		return nil, repo.ErrEmptyRef
	}
	ref, err := repo.ParseDatasetRef(path)
	if err != nil {
		return nil, fmt.Errorf("'%s' is not a valid dataset reference", path)
	}
	err = repo.CanonicalizeDatasetRef(r, &ref)
	if err != nil {
		if err != repo.ErrNoHistory || !allowFSI {
			return nil, err
		}
	}
	return &ref, nil
}
