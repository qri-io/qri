package collection

import (
	"context"
	"errors"

	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

// MigrateRepoStoreToLocalCollectionSet constructs a local collection.Set from
// legacy repo data
func MigrateRepoStoreToLocalCollectionSet(ctx context.Context, s Set, r repo.Repo) error {
	ws, ok := s.(WritableSet)
	if !ok {
		return errors.New("cannot migrate to CollectionSet. Provided CollectionSet is not writable")
	}

	datasets, err := repo.ListVersionInfoShim(r, 0, 1000000)
	if err != nil {
		return err
	}

	// empty collection migration needs to create a set for the repo owner
	if len(datasets) == 0 {
		if ls, ok := s.(*localSet); ok {
			ls.collections[r.Profiles().Owner().ID] = []dsref.VersionInfo{}
		}
	}

	book := r.Logbook()
	for i, vi := range datasets {
		ref := vi.SimpleRef()
		if _, err := book.ResolveRef(ctx, &ref); err != nil {
			log.Errorf("can't migrate dataset %s to collection. Error resolving dataset initID: %s", vi.SimpleRef(), err)
			continue
		}
		datasets[i].InitID = ref.InitID
		if ds, loadErr := dsfs.LoadDataset(ctx, r.Filesystem(), ref.Path); loadErr == nil {
			datasets[i].CommitTime = ds.Commit.Timestamp
			datasets[i].CommitTitle = ds.Commit.Title
			datasets[i].BodyRows = ds.Structure.Entries
			datasets[i].BodySize = ds.Structure.Length
			datasets[i].NumErrors = ds.Structure.ErrCount
			if ds.Meta != nil {
				datasets[i].MetaTitle = ds.Meta.Title
			}
		}
	}

	// remove any datasets that couldn't be resolved
	for i := len(datasets) - 1; i >= 0; i-- {
		if datasets[i].InitID == "" {
			datasets = append(datasets[:i], datasets[i+1:]...)
		}
	}

	return ws.Put(ctx, r.Profiles().Owner().ID, datasets...)
}
