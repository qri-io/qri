package collection

import (
	"context"
	"fmt"
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
			return nil, fmt.Errorf("resolving dataset initID: %w", err)
		}
		datasets[i].InitID = ref.InitID
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
