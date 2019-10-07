package base

import (
	"context"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/dsref"
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

// ConstructDatasetLogFromHistory constructs a log for a name if one doesn't
// exist.
func ConstructDatasetLogFromHistory(ctx context.Context, r repo.Repo, ref dsref.Ref) error {
	refs, err := DatasetLog(ctx, r, repo.DatasetRef{ Peername: ref.Username, Name: ref.Name}, 1000000, 0, true)
	if err != nil {
		return err
	}
	history := make([]*dataset.Dataset, len(refs))
	for i, ref := range refs {
		history[i] = ref.Dataset
	}

	book := r.Logbook()
	return book.ConstructDatasetLog(ctx, ref, history)
}