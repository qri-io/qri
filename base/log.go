package base

import (
	"context"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// DatasetLog fetches the change version history of a dataset
func DatasetLog(ctx context.Context, r repo.Repo, ref reporef.DatasetRef, limit, offset int, loadDatasets bool) ([]dsref.VersionInfo, error) {
	if book := r.Logbook(); book != nil {
		if versions, err := book.Versions(ctx, repo.ConvertToDsref(ref), offset, limit); err == nil {
			// logs are ok with history not existing. This keeps FSI interaction behaviour consistent
			// TODO (b5) - we should consider having "empty history" be an ok state, instead of marking as an error
			if len(versions) == 0 {
				return nil, repo.ErrNoHistory
			}
			// Logbook doesn't store the CommitMessage (see infoFromOp in logbook/logbook.go), so we
			// need to load each dataset, and assign the CommitMessage field.
			for i, v := range versions {
				if v.Path != "" {
					local, err := r.Store().Has(ctx, v.Path)
					if err != nil {
						continue
					}
					if local {
						if ds, err := dsfs.LoadDataset(ctx, r.Store(), v.Path); err == nil {
							if ds.Commit != nil {
								versions[i].CommitMessage = ds.Commit.Message
							}
						}
					}
				}
			}
			return versions, nil
		}
	}

	rlog, err := DatasetLogFromHistory(ctx, r, ref, offset, limit, loadDatasets)
	if err != nil {
		return nil, err
	}
	items := make([]dsref.VersionInfo, len(rlog))
	for i, vref := range rlog {
		items[i] = reporef.ConvertToVersionInfo(&vref)
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
// TODO(dlong): Convert to use dsref.Ref (for input) and dsref.VersionInfo (for output)
func DatasetLogFromHistory(ctx context.Context, r repo.Repo, ref reporef.DatasetRef, offset, limit int, loadDatasets bool) (rlog []reporef.DatasetRef, err error) {
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

	versions := make(chan reporef.DatasetRef)
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
	repoRef := reporef.DatasetRef{Peername: ref.Username, Name: ref.Name, Path: ref.Path}
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
