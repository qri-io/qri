package base

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

// DatasetLog fetches the history of changes to a dataset, if loadDatasets is true, dataset information will be populated
func DatasetLog(ctx context.Context, r repo.Repo, ref repo.DatasetRef, limit, offset int, loadDatasets bool) (rlog []repo.DatasetRef, err error) {
	// TODO (b5) - this is a horrible hack to handle long-lived requests when connected to IPFS
	// if we don't have the dataset locally, this process will take longer than 700 mill, because it'll
	// reach out onto the d.web to attempt to resolve previous hashes. capping the duration
	// yeilds quick results. The proper way to solve this is to feed a local-only IPFS store to
	// this entire function
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*700)
	defer cancel()

	versions := make(chan repo.DatasetRef)
	done := make(chan struct{})
	go func() {
		for {
			var ds *dataset.Dataset
			if loadDatasets {
				if ds, err = dsfs.LoadDataset(ctx, r.Store(), ref.Path); err != nil {
					return
				}
			} else {
				if ds, err = dsfs.LoadDatasetRefs(ctx, r.Store(), ref.Path); err != nil {
					return
				}
			}
			ref.Dataset = ds

			if offset <= 0 {
				// rlog = append(rlog, ref)
				versions <- ref

				limit--
				if limit == 0 {
					break
				}
			}
			if ref.Dataset.PreviousPath == "" {
				break
			}
			ref.Path = ref.Dataset.PreviousPath
			offset--
		}
		done <- struct{}{}
	}()

	for {
		select {
		case ref := <-versions:
			rlog = append(rlog, ref)
		case <-done:
			return rlog, nil
		case <-ctx.Done():
			// TODO (b5) - ths is technially a failure, handle it!
			return rlog, nil
		}
	}
}

// LogDiffResult is the result of comparing a set of references
type LogDiffResult struct {
	Head        repo.DatasetRef
	Add, Remove []repo.DatasetRef
}

// LogDiff determines the difference between an input slice of references
func LogDiff(ctx context.Context, r repo.Repo, a []repo.DatasetRef) (ldr LogDiffResult, err error) {
	if len(a) < 1 {
		return ldr, fmt.Errorf("no references provided for diffing")
	}

	alias := repo.DatasetRef{Peername: a[0].Peername, Name: a[0].Name}
	ldr.Head, err = r.GetRef(alias)
	if err != nil {
		return ldr, err
	}

	// TODO - deal with max limit / offset / pagination issuez
	b, err := DatasetLog(ctx, r, ldr.Head, 10000, 0, false)
	if err != nil {
		return ldr, err
	}

	ldr.Add, ldr.Remove = refDiff(a, b)

	return ldr, nil
}

// refDiff returns a set of additions and removals needed to sync slice a to b
func refDiff(a, b []repo.DatasetRef) (add, remove []repo.DatasetRef) {
	var present bool
	for _, aRef := range a {
		present = false
		for _, bRef := range b {
			if aRef.Equal(bRef) {
				present = true
				break
			}
		}
		if !present {
			remove = append(remove, aRef)
		}
	}

	for _, bRef := range b {
		present = false
		for _, aRef := range a {
			if bRef.Equal(aRef) {
				present = true
				break
			}
		}
		if !present {
			add = append(add, bRef)
		}
	}
	return
}
