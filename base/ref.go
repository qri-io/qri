package base

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// InLocalNamespace checks if a dataset ref is local, assumes the reference is
// already resolved
func InLocalNamespace(r repo.Repo, ref dsref.Ref) bool {
	p, err := r.Profile()
	if err != nil {
		return false
	}

	return p.ID.String() == ref.ProfileID
}

// SetPublishStatus updates the Published field of a dataset ref
func SetPublishStatus(r repo.Repo, ref dsref.Ref, published bool) error {
	if !InLocalNamespace(r, ref) {
		return fmt.Errorf("can't publish datasets that are not in your namespace")
	}

	vi := dsref.NewVersionInfoFromRef(ref)
	vi.Published = published
	return repo.PutVersionInfoShim(r, &vi)
}

// ReplaceRefIfMoreRecent replaces the given ref in the ref store, if
// it is more recent then the ref currently in the refstore
func ReplaceRefIfMoreRecent(r repo.Repo, prev, curr *reporef.DatasetRef) error {
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

// RenameDatasetRef changes a dataset's pretty name by modifying the ref in the repository
// refstore, and returns a versionInfo describing the resulting dataset reference.
func RenameDatasetRef(ctx context.Context, r repo.Repo, ref dsref.Ref, newName string) (*dsref.VersionInfo, error) {
	if !dsref.IsValidName(newName) {
		return nil, dsref.ErrDescribeValidName
	}
	if ref.Name == newName {
		return nil, fmt.Errorf("cannot rename without changing the dataset names")
	}

	next := dsref.Ref{Username: ref.Username, Name: newName}
	// Resolve the next reference to make sure it doesn't exist
	if _, newRefErr := r.ResolveRef(ctx, &next); newRefErr == nil {
		// successful resolution on rename is an error
		return nil, fmt.Errorf("dataset %q already exists", next.Human())
	} else if errors.Is(newRefErr, dsref.ErrRefNotFound) {
		// this is a good thing.
	} else {
		log.Debug(newRefErr.Error())
		return nil, fmt.Errorf("error with new reference: %w", newRefErr)
	}

	err := r.Logbook().WriteDatasetRename(ctx, ref.InitID, newName)
	if err != nil && err != logbook.ErrNoLogbook {
		return nil, err
	}

	// use the versionInfo returned from the delete to preserve fields like
	// FSIPath when replaced
	vi, err := repo.DeleteVersionInfoShim(r, ref)
	if err != nil {
		return nil, err
	}
	vi.InitID = ref.InitID
	vi.Name = newName
	err = repo.PutVersionInfoShim(r, vi)
	return vi, err
}

// ModifyRepoUsername performs all tasks necessary to switch a username
// (formerly: peername) for a local repo, and must be called when a username is
// changed
// TODO (b5) - make this transactional
func ModifyRepoUsername(ctx context.Context, r repo.Repo, book *logbook.Book, from, to string) error {
	log.Debugf("change peername: %s -> %s", from, to)
	// TODO (b5) - we need to immediately update all dataset references in the refstore on rename
	// because we currently rely on dsref as our source of canonicalization.
	// Many places in our codebase call repo.CanonicalizeDatasetRef with an alias reference
	// before doing anything, which means if the reference there is off, Canonicalize will overwrite
	// with the prior, incorrect name, and cause not-found errors in place that are properly tracking
	// updates, (like logbook). This is hacky & ugly, but helps us understand how to redesign dsrefs
	if refs, err := r.References(0, 10000000); err == nil {
		for _, ref := range refs {
			if ref.Peername == from {
				update := reporef.DatasetRef{
					Peername:  to,
					Name:      ref.Name,
					Path:      ref.Path,
					ProfileID: ref.ProfileID,
					FSIPath:   ref.FSIPath,
				}

				if err = r.DeleteRef(ref); err != nil {
					return err
				}
				if err = r.PutRef(update); err != nil {
					return err
				}
			}
		}
	}

	// we also need to update the logbook
	return book.WriteAuthorRename(ctx, to)
}
