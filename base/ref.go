package base

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qri/logbook"
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

// ReplaceRefIfMoreRecent replaces the given ref in the ref store, if
// it is more recent then the ref currently in the refstore
func ReplaceRefIfMoreRecent(r repo.Repo, prev, curr *repo.DatasetRef) error {
	var (
		prevTime time.Time
		currTime time.Time
	)
	if curr == nil || curr.Dataset == nil || curr.Dataset.Commit == nil {
		return fmt.Errorf("added dataset ref is not fully dereferenced")
	}
	currTime = curr.Dataset.Commit.Timestamp
	if prev == nil || prev.Dataset == nil || prev.Dataset.Commit == nil {
		return fmt.Errorf("previous dataset ref is not fully derefernced")
	}
	prevTime = prev.Dataset.Commit.Timestamp

	if prevTime.Before(currTime) || prevTime.Equal(currTime) {
		if err := r.PutRef(*curr); err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error putting dataset name in repo: %s", err.Error())
		}
	}
	return nil
}

// ModifyDatasetRef alters a reference by changing what dataset it refers to
func ModifyDatasetRef(ctx context.Context, r repo.Repo, current, new *repo.DatasetRef, isRename bool) (err error) {
	if err := validate.ValidName(new.Name); err != nil {
		return err
	}
	if err := repo.CanonicalizeDatasetRef(r, current); err != nil && err != repo.ErrNoHistory {
		log.Debug(err.Error())
		return fmt.Errorf("error with existing reference: %s", err.Error())
	}
	// attempt to canonicalize the new reference
	err = repo.CanonicalizeDatasetRef(r, new)
	if err == nil && isRename {
		// successful canonicalization on rename is an error
		return fmt.Errorf("dataset '%s/%s' already exists", new.Peername, new.Name)
	} else if err != repo.ErrNotFound {
		log.Debug(err.Error())
		return fmt.Errorf("error with new reference: %s", err.Error())
	}
	if isRename {
		new.Path = current.Path

		if err = r.Logbook().WriteDatasetRename(ctx, repo.ConvertToDsref(*current), new.Name); err != nil && err != logbook.ErrNoLogbook {
			return err
		}
	}
	new.FSIPath = current.FSIPath
	if err = r.DeleteRef(*current); err != nil {
		return err
	}
	if err = r.PutRef(*new); err != nil {
		return err
	}

	return nil
}
