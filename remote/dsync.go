package remote

import (
	"fmt"
	"context"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/dag"
	"github.com/qri-io/dag/dsync"
)

// DsyncConfigFunc takes a qri configuration & returns a function that
// configures dsync accordingly
func DsyncConfigFunc(cfg *config.Config, node *p2p.QriNode) func(*dsync.Config) {
	return func(dsyncConfig *dsync.Config) {
		if cfg.API.RemoteMode {
			if host := node.Host(); host != nil {
				dsyncConfig.Libp2pHost = node.Host()
			}

			dsyncConfig.PreCheck = newPreCheckHook(cfg, node)
			dsyncConfig.FinalCheck = newFinalCheckHook(cfg, node)
			dsyncConfig.OnComplete = newOnCompleteHook(cfg, node)
		}
	}
}

func newPreCheckHook(cfg *config.Config, node *p2p.QriNode) dsync.Hook {
	return func(ctx context.Context, info dag.Info, meta map[string]string) error {
		// TODO(dlong): Customization for how to decide to accept the dataset.
	if cfg.API.RemoteAcceptSizeMax == 0 {
		return fmt.Errorf("not accepting any datasets")
	}

	// If size is -1, accept any size of dataset. Otherwise, check if the size is allowed.
	if cfg.API.RemoteAcceptSizeMax != -1 {
		var totalSize uint64
		for _, s := range info.Sizes {
			totalSize += s
		}

		if totalSize >= uint64(cfg.API.RemoteAcceptSizeMax) {
			return fmt.Errorf("dataset size too large")
		}
	}

		return nil 
	}
}

func newFinalCheckHook(cfg *config.Config, node *p2p.QriNode) dsync.Hook {
	return func(ctx context.Context, info dag.Info, meta map[string]string) error {
		return nil 
	}
}

func newOnCompleteHook(cfg *config.Config, node *p2p.QriNode) dsync.Hook {
	// TODO (b5) - any error return should probs unpin the dataset?
	return func(ctx context.Context, info dag.Info, meta map[string]string) error {
		pid, err := profile.IDB58Decode(meta["profileId"])
		if err != nil {
			return err
		}

		ref := &repo.DatasetRef{
			Peername: meta["peername"],
			Name: meta["name"],
			ProfileID: pid,
			Path: meta["path"],
		}

		if err := repo.CanonicalizeDatasetRef(node.Repo, ref); err != nil {
			if err == repo.ErrNotFound {
				err = nil
			} else {
				return err
			}
		}

		// add completed pushed dataset to our refs
		// TODO (b5) - this could overwrite any FSI links & other ref details
		// need to investigate
		return node.Repo.PutRef(*ref)
	}
}
