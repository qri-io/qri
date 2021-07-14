package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/buildrepo"
)

// MigrateRepoStoreToLocalCollectionSet constructs a local collection.Set from
// legacy repo data
func MigrateRepoStoreToLocalCollectionSet(ctx context.Context, repoDir string, cfg *config.Config) error {
	if _, err := os.Stat(filepath.Join(repoDir, "collections")); !os.IsNotExist(err) {
		// skip migration if collection dir exsists
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	r, err := buildrepo.New(ctx, repoDir, cfg)
	if err != nil {
		return fmt.Errorf("constructing repo: %w", err)
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

	s, err := collection.NewLocalSet(ctx, event.NilBus, repoDir)
	if err != nil {
		return err
	}

	ws := s.(collection.WritableSet)
	if err = ws.Put(ctx, r.Profiles().Owner().ID, datasets...); err != nil {
		return err
	}

	cancel()
	<-r.Done()
	return nil
}
