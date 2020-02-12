package base

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// RemoveEntireDataset removes all of the information in the repository about a dataset. It will
// continue on even if some error occurs, returning a comma-separated list of what locations data
// was removed from, as well the last error that occured, if any.
// Note that in particular, FSI is not handled at all by this function. Callers should also
// call any relevent FSI operations.
func RemoveEntireDataset(ctx context.Context, r repo.Repo, ref dsref.Ref, history []dsref.VersionInfo) (didRemove string, removeErr error) {
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
	if err := book.WriteDatasetDelete(ctx, ref); err == nil {
		didRemove = appendString(didRemove, "logbook")
	} else {
		// If the logbook is missing, it's not an error worth stopping for, since we're
		// deleting the dataset anyway. This can happen from adding a foreign dataset.
		if err != oplog.ErrNotFound {
			log.Debugf("Remove, logbook.WriteDatasetDelete failed, error: %s", err)
			removeErr = err
		}
	}
	// remove the ref from the ref store
	datasetRef := reporef.DatasetRef{
		Peername:  ref.Username,
		Name:      ref.Name,
		Path:      ref.Path,
		ProfileID: profile.ID(ref.ProfileID),
	}
	if _, err := r.GetRef(datasetRef); err == nil {
		didRemove = appendString(didRemove, "refstore")
	}
	if err := r.DeleteRef(datasetRef); err != nil {
		log.Debugf("Remove, DeleteRef failed, error: %s", err)
		removeErr = err
	}
	return didRemove, removeErr
}

// RemoveNVersionsFromStore removes n versions of a dataset from the store starting with
// the most recent version
// when n == -1, remove all versions
// does not remove the dataset reference
func RemoveNVersionsFromStore(ctx context.Context, r repo.Repo, curr dsref.Ref, n int) (dsref.Ref, error) {
	var err error
	if r == nil {
		return curr, fmt.Errorf("need a repo")
	}
	if curr.Path == "" {
		return curr, fmt.Errorf("need a dataset reference with a path")
	}

	if n < -1 {
		return curr, fmt.Errorf("invalid 'n', n should be n >= 0 or n == -1 to indicate removing all versions")
	}

	// load previous dataset into prev
	ds, err := dsfs.LoadDatasetRefs(ctx, r.Store(), curr.Path)
	if err != nil {
		return curr, err
	}

	// Set a timeout for looking up the previous dataset versions.
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer timeoutCancel()

	i := n

	for i != 0 {
		// Decrement our counter. If counter was -1, this loop will continue forever, until a
		// blank PreviousPath is found.
		i--
		// unpin dataset, ignoring "not pinned" errors
		datasetRef := reporef.DatasetRef{
			Peername:  curr.Username,
			Name:      curr.Name,
			Path:      curr.Path,
			ProfileID: profile.ID(curr.ProfileID),
		}
		if err = UnpinDataset(ctx, r, datasetRef); err != nil && !strings.Contains(err.Error(), "not pinned") {
			return curr, err
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
		next, err := dsfs.LoadDatasetRefs(timeoutCtx, r.Store(), ds.PreviousPath)
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
		curr.Path = next.Path
		ds = next
	}

	err = r.Logbook().WriteVersionDelete(ctx, curr, n)
	if err == logbook.ErrNoLogbook {
		err = nil
	}

	return curr, nil
}

// This is inefficient and not great style, use it here just as a convenience.
func appendString(first, second string) string {
	if first == "" {
		return second
	}
	return fmt.Sprintf("%s, %s", first, second)
}
