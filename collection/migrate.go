package collection

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
)

// MigrateRepoStoreToLocalCollectionSet constructs a local collection.Set from
// legacy repo data
func MigrateRepoStoreToLocalCollectionSet(ctx context.Context, bus event.Bus, repoDir string, r repo.Repo) (Set, error) {
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
