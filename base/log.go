package base

import (
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

// DatasetLog fetches the history of changes to a dataset, if loadDatasets is true, dataset information will be populated
func DatasetLog(r repo.Repo, ref repo.DatasetRef, limit, offset int, loadDatasets bool) (rlog []repo.DatasetRef, err error) {
	for {
		var ds *dataset.Dataset
		if loadDatasets {
			if ds, err = dsfs.LoadDataset(r.Store(), ref.Path); err != nil {
				return
			}
		} else {
			if ds, err = dsfs.LoadDatasetRefs(r.Store(), ref.Path); err != nil {
				return
			}
		}
		ref.Dataset = ds

		offset--
		if offset > 0 {
			continue
		}

		rlog = append(rlog, ref)

		limit--
		if limit == 0 || ref.Dataset.PreviousPath == "" {
			break
		}
		ref.Path = ref.Dataset.PreviousPath
	}

	return rlog, nil
}

// LogDiffResult is the result of comparing a set of references
type LogDiffResult struct {
	Head        repo.DatasetRef
	Add, Remove []repo.DatasetRef
}

// LogDiff determines the difference between an input slice of references
func LogDiff(r repo.Repo, a []repo.DatasetRef) (ldr LogDiffResult, err error) {
	if len(a) < 1 {
		return ldr, fmt.Errorf("no references provided for diffing")
	}

	alias := repo.DatasetRef{Peername: a[0].Peername, Name: a[0].Name}
	ldr.Head, err = r.GetRef(alias)
	if err != nil {
		return ldr, err
	}

	// TODO - deal with max limit / offset / pagination issuez
	b, err := DatasetLog(r, ldr.Head, 10000, 0, false)
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
