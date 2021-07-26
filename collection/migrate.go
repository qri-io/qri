package collection

import (
	"context"
	"errors"

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

	book := r.Logbook()
	for i, vi := range datasets {
		ref := vi.SimpleRef()
		if _, err := book.ResolveRef(ctx, &ref); err != nil {
			log.Warnf("can't migrate dataset %s to collection. Error resolving dataset initID: %s", vi.SimpleRef(), err)
			continue
		}
		datasets[i].InitID = ref.InitID
	}

	// remove any datasets that couldn't be resolved
	for i := len(datasets) - 1; i >= 0; i-- {
		if datasets[i].InitID == "" {
			datasets = append(datasets[:i], datasets[i+1:]...)
		}
	}

	return ws.Put(ctx, r.Profiles().Owner().ID, datasets...)
}
