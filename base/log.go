package base

import (
	"context"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

// DatasetLogItem is a line item in a dataset response
type DatasetLogItem struct {
	Ref dsref.Ref `json:"ref,omitempty"`
	// Creation timestamp
	Timestamp time.Time `json:"timestamp,omitempty"`
	// Title field from dataset.commit component
	CommitTitle string `json:"commitTitle,omitempty"`
	// Message field from dataset.commit component
	CommitMessage string `json:"commitMessage,omitempty"`
	// Published indicates if this version has been published
	Published bool `json:"published,omitempty"`
	// Size of dataset in bytes
	Size uint64 `json:"size,omitempty"`
	// Local indicates the connected filesystem has this version available
	Local bool `json:"local,omitempty"`
}

// DatasetLog fetches the change version history of a dataset
func DatasetLog(ctx context.Context, r repo.Repo, ref repo.DatasetRef, limit, offset int, loadDatasets bool) (items []DatasetLogItem, err error) {
	if book := r.Logbook(); book != nil {
		if versions, err := book.Versions(repo.ConvertToDsref(ref), offset, limit); err == nil {
			items = make([]DatasetLogItem, len(versions))

			for i, v := range versions {
				items[i] = DatasetLogItem{
					Ref:         v.Ref,
					Published:   v.Published,
					Timestamp:   v.Timestamp,
					CommitTitle: v.CommitTitle,
					Size:        v.Size,
				}

				if v.Ref.Path != "" {
					items[i].Local, err = r.Store().Has(ctx, v.Ref.Path)
					if err != nil {
						return nil, err
					}
					if items[i].Local {
						if ds, err := dsfs.LoadDataset(ctx, r.Store(), v.Ref.Path); err == nil {
							if ds.Commit != nil {
								items[i].CommitMessage = ds.Commit.Message
							}
						}
					}
				}

				i--
			}
			return items, nil
		}
	}

	rlog, err := DatasetLogFromHistory(ctx, r, ref, offset, limit, loadDatasets)
	if err != nil {
		return nil, err
	}
	items = make([]DatasetLogItem, len(rlog))
	for i, vref := range rlog {
		items[i] = DatasetLogItem{Ref: repo.ConvertToDsref(vref)}
		if vref.Dataset != nil && vref.Dataset.Commit != nil {
			items[i].Timestamp = vref.Dataset.Commit.Timestamp
			items[i].CommitTitle = vref.Dataset.Commit.Title
			items[i].CommitMessage = vref.Dataset.Commit.Message
		}
	}

	// add a history entry b/c we didn't have one, but repo didn't error
	if pro, err := r.Profile(); err == nil && ref.Peername == pro.Peername {
		go func() {
			if err := constructDatasetLogFromHistory(context.Background(), r, repo.ConvertToDsref(ref)); err != nil {
				log.Errorf("constructDatasetLogFromHistory: %s", err)
			}
		}()
	}

	return items, err
}

// DatasetLogFromHistory fetches the history of changes to a dataset by walking
// backwards through dataset commits. if loadDatasets is true, dataset
// information will be populated
func DatasetLogFromHistory(ctx context.Context, r repo.Repo, ref repo.DatasetRef, offset, limit int, loadDatasets bool) (rlog []repo.DatasetRef, err error) {
	if err := repo.CanonicalizeDatasetRef(r, &ref); err != nil {
		return nil, err
	}

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

// constructDatasetLogFromHistory constructs a log for a name if one doesn't
// exist.
func constructDatasetLogFromHistory(ctx context.Context, r repo.Repo, ref dsref.Ref) error {
	repoRef := repo.DatasetRef{Peername: ref.Username, Name: ref.Name, Path: ref.Path}
	refs, err := DatasetLogFromHistory(ctx, r, repoRef, 0, 1000000, true)
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
