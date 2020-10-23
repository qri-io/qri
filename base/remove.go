package base

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
)

// RemoveEntireDataset removes all of the information in the repository about a dataset. It will
// continue on even if some error occurs, returning a comma-separated list of what locations data
// was removed from, as well the last error that occurred, if any.
// Note that in particular, FSI is not handled at all by this function. Callers should also
// call any relevent FSI operations.
func RemoveEntireDataset(ctx context.Context, r repo.Repo, ref dsref.Ref, history []DatasetLogItem) (didRemove string, removeErr error) {
	// If the dataset has no history (such as running `qri init` without `qri save`), then
	// the ref has no path. Can't call RemoveNVersionsFromStore without a path, but don't
	// need to call it anyway. Skip it.
	if len(history) > 0 {
		// Delete entire dataset for all generations.
		if _, err := RemoveNVersionsFromStore(ctx, r, ref, -1); err == nil {
			didRemove = appendString(didRemove, "history")
		} else {
			log.Debugf("Remove, base.RemoveNVersionsFromStore failed, error: %s", err)
			removeErr = err
		}
	}
	// Write the deletion to the logbook.
	book := r.Logbook()
	// TODO(dustmop): When we switch to initIDs, use the initID passed to this function, retrieved
	// from the top-level resolver.
	initID, err := book.RefToInitID(ref)
	if err != nil && err != logbook.ErrNoLogbook {
		log.Debugf("Remove, logbook.RefToInitID failed, error: %s", err)
		removeErr = err
	}
	if ref.Username == book.Username() {
		// TOOD(dustmop): Logbook should validate the fact that author's should only be able to
		// write to their own logs. Trying to write to another user's log should throw an error.
		if err := book.WriteDatasetDelete(ctx, initID); err == nil {
			didRemove = appendString(didRemove, "logbook")
		} else {
			log.Debugf("Remove, logbook.WriteDatasetDelete failed, error: %s", err)
			removeErr = err
		}
	} else {
		if err := book.RemoveLog(ctx, ref); err == nil {
			didRemove = appendString(didRemove, "logbook")
		} else {
			log.Debugf("Remove, logbook.RemoveLog failed, error: %s", err)
			removeErr = err
		}
	}
	// remove the ref from the ref store
	if _, err := repo.GetVersionInfoShim(r, ref); err == nil {
		didRemove = appendString(didRemove, "refstore")
	}
	if _, err := repo.DeleteVersionInfoShim(r, ref); err != nil {
		log.Debugf("Remove, DeleteRef failed, error: %s", err)
		removeErr = err
	}
	return didRemove, removeErr
}

// RemoveNVersionsFromStore removes n versions of a dataset from the store starting with
// the most recent version
// when n == -1, remove all versions
// does not remove the dataset reference
func RemoveNVersionsFromStore(ctx context.Context, r repo.Repo, curr dsref.Ref, n int) (*dsref.VersionInfo, error) {
	var err error
	if r == nil {
		return nil, fmt.Errorf("need a repo")
	}
	if curr.Path == "" {
		return nil, fmt.Errorf("need a dataset reference with a path")
	}
	fs := r.Filesystem()

	if n < -1 {
		return nil, fmt.Errorf("invalid 'n', n should be n >= 0 or n == -1 to indicate removing all versions")
	}

	// load previous dataset into prev
	ds, err := dsfs.LoadDatasetRefs(ctx, fs, curr.Path)
	if err != nil {
		return nil, err
	}

	// Set a timeout for looking up the previous dataset versions.
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer timeoutCancel()

	// Copy the dsref, modify it as we step back through the dataset's history.
	dest := curr.Copy()
	i := n

	for i != 0 {
		// Decrement our counter. If counter was -1, this loop will continue forever, until a
		// blank PreviousPath is found.
		i--
		// unpin dataset, ignoring "not pinned" errors
		if err = UnpinDataset(ctx, r, dest.Path); err != nil && !strings.Contains(err.Error(), "not pinned") {
			return nil, err
		}
		// if no previous path, break
		if ds.PreviousPath == "" {
			break
		}
		// Load previous dataset into prev. Use a timeout on this lookup, because we don't want
		// IPFS to hang if it's not available, since that would only happen if it's not local to
		// our machine. This situation can occur if we have added a foreign dataset from the
		// registry, and don't have previous versions. In situations like that, it's okay to
		// break this loop since the only purpose of loading previous versions is to unpin them;
		// if we don't have all previous versions, we probably don't have them pinned.
		// TODO(dlong): If IPFS gains the ability to ask "do I have these blocks locally", use
		// that instead of the network-aware LoadDatasetRefs.
		loadedPrev, err := dsfs.LoadDatasetRefs(timeoutCtx, fs, ds.PreviousPath)
		if err != nil {
			// Note: We want delete to succeed even if datasets are remote, so we don't fail on
			// this error, and break early instead.
			if strings.Contains(err.Error(), "context deadline exceeded") {
				log.Debugf("could not load dataset ref, not found locally")
				break
			}
			// TODO (b5) - removing dataset versions should rely on logbook, which is able
			// to traverse across missing datasets in qfs
			log.Debugf("error fetching previous: %s", err)
			break
		}
		dest.Path = loadedPrev.Path
		ds = loadedPrev
	}

	info, err := RewindDatasetRef(ctx, r, curr, dest)
	if err != nil {
		return nil, err
	}

	// TODO(dustmop): When we switch to initIDs, use the initID passed to this function, retrieved
	// from the top-level resolver.
	initID, err := r.Logbook().RefToInitID(curr)
	if err == logbook.ErrNoLogbook || err == logbook.ErrNotFound {
		// If logbook doesn't exist or doesn't know about this dataset, it's not an error, since
		// we're just trying to remove it. Return successfully.
		return info, nil
	}
	if err = r.Logbook().WriteVersionDelete(ctx, initID, n); err != nil {
		return info, err
	}

	return info, nil
}

// This is inefficient and not great style, use it here just as a convenience.
func appendString(first, second string) string {
	if first == "" {
		return second
	}
	return fmt.Sprintf("%s, %s", first, second)
}

// RewindDatasetRef changes a dsref in the repository refstore, by setting the path back to
// an older value, and returns a versionInfo describing the resulting dataset reference.
func RewindDatasetRef(ctx context.Context, r repo.Repo, curr, next dsref.Ref) (*dsref.VersionInfo, error) {
	if !dsref.IsValidName(next.Name) {
		return nil, dsref.ErrDescribeValidName
	}
	if curr.Username != next.Username || curr.ProfileID != next.ProfileID {
		return nil, fmt.Errorf("cannot change username or profileID of a dataset")
	}
	if curr.Name != next.Name {
		return nil, fmt.Errorf("cannot rewind dataset ref with a different name")
	}

	currVi, err := repo.GetVersionInfoShim(r, curr)
	if err != nil && err != repo.ErrNoHistory {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error with existing reference: %s", err.Error())
	}

	nextVi, err := repo.GetVersionInfoShim(r, next)
	if err != nil && err != repo.ErrNoHistory {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error with existing reference: %s", err.Error())
	}

	if _, err := repo.DeleteVersionInfoShim(r, currVi.SimpleRef()); err != nil {
		return nil, err
	}

	nextVi.Path = next.Path
	if err := repo.PutVersionInfoShim(r, nextVi); err != nil {
		return nil, err
	}
	return nextVi, nil
}
