package base

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// InLocalNamespace checks if a dataset ref is local, assumes the reference is already canonicalized
func InLocalNamespace(r repo.Repo, ref *reporef.DatasetRef) bool {
	p, err := r.Profile()
	if err != nil {
		return false
	}

	return p.ID == ref.ProfileID
}

// SetPublishStatus updates the Published field of a dataset ref
func SetPublishStatus(r repo.Repo, ref *reporef.DatasetRef, published bool) error {
	if !InLocalNamespace(r, ref) {
		return fmt.Errorf("can't publish datasets that are not in your namespace")
	}

	ref.Published = published
	return r.PutRef(*ref)
}

// ToDatasetRef parses the dataset ref and looks it up in the refstore, allows refs with no history
// TODO(dustmop): In a future change, remove the third parameter from this function
func ToDatasetRef(path string, r repo.Repo, _ bool) (*reporef.DatasetRef, error) {
	if path == "" {
		return nil, repo.ErrEmptyRef
	}
	ref, err := repo.ParseDatasetRef(path)
	if err != nil {
		return nil, fmt.Errorf("'%s' is not a valid dataset reference", path)
	}
	err = repo.CanonicalizeDatasetRef(r, &ref)
	if err != nil && err != repo.ErrNoHistory {
		return nil, err
	}
	return &ref, nil
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

// ModifyDatasetRef changes a dsref in the repository refstore, either name or path, and returns
// a versionInfo describing the resulting dataset reference. Use for renaming a dataset's pretty
// name or for rewinding to before some recent number of commits.
func ModifyDatasetRef(ctx context.Context, r repo.Repo, curr, next dsref.Ref) (*dsref.VersionInfo, error) {
	if !dsref.IsValidName(next.Name) {
		return nil, dsref.ErrDescribeValidName
	}
	if curr.Username != next.Username || curr.ProfileID != next.ProfileID {
		return nil, fmt.Errorf("cannot change username or profileID of a dataset")
	}

	currProfileID, err := profile.IDB58Decode(curr.ProfileID)
	if err != nil {
		if curr.ProfileID == "" {
			currProfileID = profile.IDRawByteString("")
		} else {
			return nil, err
		}
	}

	nextProfileID, err := profile.IDB58Decode(next.ProfileID)
	if err != nil {
		if next.ProfileID == "" {
			nextProfileID = profile.IDRawByteString("")
		} else {
			return nil, err
		}
	}

	currRef := reporef.DatasetRef{
		Peername:  curr.Username,
		ProfileID: currProfileID,
		Name:      curr.Name,
		Path:      curr.Path,
	}
	nextRef := reporef.DatasetRef{
		Peername:  next.Username,
		ProfileID: nextProfileID,
		Name:      next.Name,
		Path:      next.Path,
	}
	isRename := false

	if currRef.Name != nextRef.Name {
		// Renaming the pretty name of a dataset
		isRename = true
		if currRef.Path != "" || nextRef.Path != "" || currRef.ProfileID != "" || nextRef.ProfileID != "" {
			return nil, fmt.Errorf("can only rename using references that are human-friendly")
		}
		// Canonicalize the existing reference so that we have ProfileID and Path
		if err := repo.CanonicalizeDatasetRef(r, &currRef); err != nil && err != repo.ErrNoHistory {
			log.Debug(err.Error())
			return nil, fmt.Errorf("error with existing reference: %s", err.Error())
		}
		// Canonicalize the next reference to make sure it doesn't exist
		err := repo.CanonicalizeDatasetRef(r, &nextRef)
		if err == nil {
			// successful canonicalization on rename is an error
			return nil, fmt.Errorf("dataset '%s/%s' already exists", nextRef.Peername, nextRef.Name)
		} else if err != repo.ErrNotFound {
			log.Debug(err.Error())
			return nil, fmt.Errorf("error with new reference: %s", err.Error())
		}
		// Assign state that stays the same during a rename
		nextRef.ProfileID = currRef.ProfileID
		nextRef.Path = currRef.Path
		nextRef.FSIPath = currRef.FSIPath
	} else {
		// Names are the same, deleting some number of commits
		if currRef.Path == nextRef.Path {
			return nil, fmt.Errorf("cannot modify ref, no changes found")
		}
		// Both references need to canonicalize
		if err := repo.CanonicalizeDatasetRef(r, &currRef); err != nil && err != repo.ErrNoHistory {
			log.Debug(err.Error())
			return nil, fmt.Errorf("error with existing reference: %s", err.Error())
		}
		if err := repo.CanonicalizeDatasetRef(r, &nextRef); err != nil && err != repo.ErrNoHistory {
			log.Debug(err.Error())
			return nil, fmt.Errorf("error with target reference: %s", err.Error())
		}
	}

	// Copy data back to the dsref after canonicalization.
	// TODO(dlong): Once we fully convert to dsref this will be unnecessary
	curr.Username = currRef.Peername
	curr.ProfileID = currRef.ProfileID.String()

	if isRename {
		err := r.Logbook().WriteDatasetRename(ctx, curr, nextRef.Name)
		if err != nil && err != logbook.ErrNoLogbook {
			return nil, err
		}
	}
	if err := r.DeleteRef(currRef); err != nil {
		return nil, err
	}
	if err := r.PutRef(nextRef); err != nil {
		return nil, err
	}

	return &dsref.VersionInfo{
		Username:  nextRef.Peername,
		ProfileID: nextRef.ProfileID.String(),
		Name:      nextRef.Name,
		Path:      nextRef.Path,
		FSIPath:   nextRef.FSIPath,
	}, nil
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
