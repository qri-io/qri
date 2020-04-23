// Package buildrepo initializes a qri repo
package buildrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	ipfs "github.com/qri-io/qfs/cafs/ipfs"
	"github.com/qri-io/qfs/cafs/ipfs_http"
	"github.com/qri-io/qfs/httpfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
	"github.com/qri-io/qri/repo/profile"
)

var (
	pluginLoadLock  sync.Once
	pluginLoadError error
)

// LoadIPFSPluginsOnce runs IPFS plugin initialization.
// we need to load plugins before attempting to configure IPFS, flatfs is
// specified as part of the default IPFS configuration, but all flatfs
// code is loaded as a plugin.  ¯\_(ツ)_/¯
//
// This works without anything present in the /.ipfs/plugins/ directory b/c
// the default plugin set is complied into go-ipfs (and subsequently, the
// qri binary) by default
func LoadIPFSPluginsOnce(path string) error {
	body := func() {
		pluginLoadError = ipfs.LoadPlugins(path)
	}
	pluginLoadLock.Do(body)
	return pluginLoadError
}

// New creates storage a qri repo can use for configuration
// we're still carrying around a legacy cafs.Filestore, but one day should be
// able to get down to just a qfs.Filesystem return
func New(ctx context.Context, path string, cfg *config.Config) (repo.Repo, error) {
	store, err := NewCAFSStore(ctx, cfg)
	if err != nil {
		return nil, err
	}

	fs, err := NewFilesystem(cfg, store)
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

		cache, err := newDscache(ctx, fs, book, path)
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
func NewFilesystem(cfg *config.Config, store cafs.Filestore) (qfs.Filesystem, error) {
	mux := map[string]qfs.Filesystem{
		"local": localfs.NewFS(),
		"http":  httpfs.NewFS(),
		"cafs":  store,
	}

	if ipfss, ok := store.(*ipfs.Filestore); ok {
		mux["ipfs"] = ipfss
	}

	fsys := qfs.NewMux(mux)
	return fsys, nil
}

// NewCAFSStore creates a cafs.Filestore store from configuration
// we're in the process of absorbing cafs.Filestore into qfs.Filesystem, use
// a qfs.Filesystem instead
func NewCAFSStore(ctx context.Context, cfg *config.Config) (store cafs.Filestore, err error) {
	switch cfg.Store.Type {
	case "ipfs":
		path := cfg.Store.Path
		if path == "" && os.Getenv("IPFS_PATH") != "" {
			path = os.Getenv("IPFS_PATH")
		} else if path == "" {
			home, err := homedir.Dir()
			if err != nil {
				return nil, fmt.Errorf("creating IPFS store: %s", err)
			}
			path = filepath.Join(home, ".ipfs")
		}

		if err := LoadIPFSPluginsOnce(path); err != nil {
			return nil, err
		}

		fsOpts := []ipfs.Option{
			func(c *ipfs.StoreCfg) {
				c.Ctx = ctx
				c.FsRepoPath = path
			},
			ipfs.OptsFromMap(cfg.Store.Options),
		}
		return ipfs.NewFilestore(fsOpts...)
	case "ipfs_http":
		urli, ok := cfg.Store.Options["url"]
		if !ok {
			return nil, fmt.Errorf("ipfs_http store requires 'url' option")
		}
		urlStr, ok := urli.(string)
		if !ok {
			return nil, fmt.Errorf("ipfs_http 'url' option must be a string")
		}
		return ipfs_http.New(urlStr)
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

func newDscache(ctx context.Context, fs qfs.Filesystem, book *logbook.Book, repoPath string) (*dscache.Dscache, error) {
	// This seems to be a bug, the repoPath does not end in "qri" in some tests.
	if !strings.HasSuffix(repoPath, "qri") {
		return nil, fmt.Errorf("invalid repo path")
	}
	dscachePath := filepath.Join(repoPath, "dscache.qfb")
	return dscache.NewDscache(ctx, fs, book, dscachePath), nil
}
