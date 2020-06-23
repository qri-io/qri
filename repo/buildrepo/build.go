// Package buildrepo initializes a qri repo
package buildrepo

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qfs/qipfs/qipfs_http"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/event/hook"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	fsrepo "github.com/qri-io/qri/repo/fs"
	"github.com/qri-io/qri/repo/profile"
)

// New is the canonical method for building a repo
func New(ctx context.Context, path string, cfg *config.Config) (repo.Repo, error) {
	fs, err := NewFilesystem(ctx, cfg)
	if err != nil {
		return nil, err
	}

	store, err := NewCAFSStore(cfg, fs)
	if err != nil {
		return nil, err
	}

	pro, err := profile.NewProfile(cfg.Profile)
	if err != nil {
		return nil, err
	}

	switch cfg.Repo.Type {
	case "fs":
		book, err := newLogbook(fs, pro, path)
		if err != nil {
			return nil, err
		}

		cache, err := newDscache(ctx, fs, book, pro.Peername, path)
		if err != nil {
			return nil, err
		}

		r, err := fsrepo.NewRepo(store, fs, book, cache, pro, path)
		return r, err
	case "mem":
		return repo.NewMemRepo(pro, store, fs, profile.NewMemStore())
	default:
		return nil, fmt.Errorf("unknown repo type: %s", cfg.Repo.Type)
	}
}

// NewFilesystem creates a qfs.Filesystem from configuration
func NewFilesystem(ctx context.Context, cfg *config.Config) (*muxfs.Mux, error) {
	qriPath := filepath.Dir(cfg.Path())

	for i, fsCfg := range cfg.Filesystems {
		if fsCfg.Type == "ipfs" {
			if path, ok := fsCfg.Config["path"].(string); ok {
				if !filepath.IsAbs(path) {
					// resolve relative filepaths
					cfg.Filesystems[i].Config["path"] = filepath.Join(qriPath, path)
				}
			}
		}
	}

	if cfg.Repo.Type == "mem" {
		hasMemType := false
		for _, fsCfg := range cfg.Filesystems {
			if fsCfg.Type == "mem" {
				hasMemType = true
			}
		}
		if !hasMemType {
			cfg.Filesystems = append(cfg.Filesystems, qfs.Config{Type: "mem"})
		}
	}

	return muxfs.New(ctx, cfg.Filesystems)
}

// NewCAFSStore creates a cafs.Filestore store from configuration
// we're in the process of absorbing cafs.Filestore into qfs.Filesystem, use
// a qfs.Filesystem instead
func NewCAFSStore(cfg *config.Config, mux *muxfs.Mux) (store cafs.Filestore, err error) {
	switch cfg.Store.Type {
	case "ipfs":
		return mux.CAFSStoreFromIPFS(), nil
	case "ipfs_http":
		urli, ok := cfg.Store.Options["url"]
		if !ok {
			return nil, fmt.Errorf("qipfs_http store requires 'url' option")
		}
		urlStr, ok := urli.(string)
		if !ok {
			return nil, fmt.Errorf("qipfs_http 'url' option must be a string")
		}
		return qipfs_http.NewFilesystem(map[string]interface{}{
			"url": urlStr,
		})
	case "map":
		return cafs.NewMapstore(), nil
	default:
		return nil, fmt.Errorf("unknown store type: %s", cfg.Store.Type)
	}
}

// TODO (b5) - if we had a better logbook constructor, this wouldn't need to exist
func newLogbook(fs qfs.Filesystem, pro *profile.Profile, repoPath string) (book *logbook.Book, err error) {
	logbookPath := filepath.Join(repoPath, "logbook.qfb")
	return logbook.NewJournal(pro.PrivKey, pro.Peername, fs, logbookPath)
}

func newDscache(ctx context.Context, fs qfs.Filesystem, book *logbook.Book, username, repoPath string) (*dscache.Dscache, error) {
	// This seems to be a bug, the repoPath does not end in "qri" in some tests.
	if !strings.HasSuffix(repoPath, "qri") {
		return nil, fmt.Errorf("invalid repo path")
	}
	dscachePath := filepath.Join(repoPath, "dscache.qfb")
	return dscache.NewDscache(ctx, fs, []hook.ChangeNotifier{book}, username, dscachePath), nil
}
