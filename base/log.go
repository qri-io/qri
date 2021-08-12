package base

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

// TimeoutDuration is the duration allowed for a datasetLog lookup before it times out
const TimeoutDuration = 100 * time.Millisecond

// DatasetLog fetches the change version history of a dataset
func DatasetLog(ctx context.Context, r repo.Repo, ref dsref.Ref, limit, offset int, loadDatasets bool) ([]dsref.VersionInfo, error) {
	if book := r.Logbook(); book != nil {
		if items, err := book.Items(ctx, ref, offset, limit); err == nil {
			// logs are ok with history not existing. This keeps FSI interaction behaviour consistent
			// TODO (b5) - we should consider having "empty history" be an ok state, instead of marking as an error
			if len(items) == 0 {
				return nil, repo.ErrNoHistory
			}
			// Logbook doesn't store the CommitMessage and CommitTitle
			// (see infoFromOp in logbook/logbook.go), so we need to load
			// each dataset, and assign the CommitMessage and CommitTitle field.
			for i, item := range items {
				if item.Path != "" {
					local, err := r.Filesystem().Has(ctx, item.Path)
					if err != nil {
						continue
					}
					if local {
						if ds, err := dsfs.LoadDataset(ctx, r.Filesystem(), item.Path); err == nil {
							if ds.Commit != nil {
								items[i].CommitMessage = ds.Commit.Message
							}
						}
					}
					items[i].Foreign = !local
				}
			}
			return items, nil
		}
	}

	if ref.Path == "" {
		return nil, fmt.Errorf("cannot build history: %w", dsref.ErrPathRequired)
	}

	datasets, err := StoredHistoricalDatasets(ctx, r, ref.Path, offset, limit, loadDatasets)
	if err != nil {
		return nil, err
	}
	items := make([]dsref.VersionInfo, len(datasets))
	for i, ds := range datasets {
		ds.Name = ref.Name
		ds.Peername = ref.Username
		ds.ProfileID = ref.ProfileID
		items[i] = dsref.ConvertDatasetToVersionInfo(ds)
	}

	// add a history entry b/c we didn't have one, but repo didn't error
	pro := r.Profiles().Owner(ctx)
	if ref.Username == pro.Peername {
		go func() {
			if err := constructDatasetLogFromHistory(context.Background(), r, ref); err != nil {
				log.Errorf("constructDatasetLogFromHistory: %s", err)
			}
		}()
	}

	return items, err
}

// StoredHistoricalDatasets fetches the history of changes to a dataset by walking
// backwards through dataset commits. if loadDatasets is true, dataset
// information will be populated
func StoredHistoricalDatasets(ctx context.Context, r repo.Repo, headPath string, offset, limit int, loadDatasets bool) (log []*dataset.Dataset, err error) {
	fs := r.Filesystem()
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutDuration)
	defer cancel()

	versions := make(chan *dataset.Dataset)
	done := make(chan struct{})
	path := headPath
	go func() {
		for {
			var ds *dataset.Dataset
			if loadDatasets {
				if ds, err = dsfs.LoadDataset(timeoutCtx, fs, path); err != nil {
					return
				}
			} else {
				if ds, err = dsfs.LoadDatasetRefs(timeoutCtx, fs, path); err != nil {
					return
				}
			}

			if offset <= 0 {
				versions <- ds
				limit--
				if limit == 0 {
					break
				}
			}
			if ds.PreviousPath == "" {
				break
			}
			path = ds.PreviousPath
			offset--
		}
		done <- struct{}{}
	}()

	for {
		select {
		case ref := <-versions:
			log = append(log, ref)
		case <-done:
			return log, nil
		case <-timeoutCtx.Done():
			return log, ErrDatasetLogTimeout
		case <-ctx.Done():
			return log, ErrDatasetLogTimeout
		}
	}
}

// constructDatasetLogFromHistory constructs a log for a name if one doesn't
// exist.
func constructDatasetLogFromHistory(ctx context.Context, r repo.Repo, ref dsref.Ref) error {
	history, err := StoredHistoricalDatasets(ctx, r, ref.Path, 0, 1000000, true)
	if err != nil {
		return err
	}

	book := r.Logbook()
	return book.ConstructDatasetLog(ctx, r.Profiles().Owner(ctx), ref, history)
}
