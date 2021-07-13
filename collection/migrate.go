package collection

import (
	"context"
	"os"
	"path/filepath"

	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
)

func MigrateRepoStoreToLocalCollectionSet(ctx context.Context, bus event.Bus, repoDir string, r repo.Repo) (Set, error) {
	if _, err := os.Stat(filepath.Join(repoDir, collectionsDirName)); !os.IsNotExist(err) {
		// skip migration if collection dir exsists
		return NewLocalSet(ctx, bus, repoDir)
	}

	datasets, err := repo.ListVersionInfoShim(r, 0, 1000000)
	if err != nil {
		return nil, err
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

	s, err := NewLocalSet(ctx, bus, repoDir)
	if err != nil {
		return nil, err
	}

	ws := s.(WritableSet)
	if err = ws.Put(ctx, r.Profiles().Owner().ID, datasets...); err != nil {
		return nil, err
	}

	return ws, nil
}
