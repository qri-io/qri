// Package buildrepo initializes a qri repo
package buildrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qfs/cafs/ipfs_http"
	"github.com/qri-io/qfs/muxfs"
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
	// TODO (ramfox): adding a default mux config
	// after the config refactor to add a `Filesystem` field, that config
	// section should replace this
	// defaults are taken from the old buildrepo.NewFilesystem func
	muxConfig := []muxfs.MuxConfig{
		{Type: "local"},
		{Type: "http"},
	}

	// TODO(ramfox): adding this to switch statement until we
	// add a Filesystem config
	switch cfg.Store.Type {
	case "ipfs":
		path := cfg.Store.Path
		// TODO (ramfox): this should change when we migrate the
		// config to default to `${QRI_PATH}/.ipfs`
		if path == "" && os.Getenv("IPFS_PATH") != "" {
			path = os.Getenv("IPFS_PATH")
		} else if path == "" {
			home, err := homedir.Dir()
			if err != nil {
				return nil, fmt.Errorf("creating IPFS store: %s", err)
			}
			path = filepath.Join(home, ".ipfs")
		}

		ipfsCfg := muxfs.MuxConfig{
			Type: "ipfs",
			Config: map[string]interface{}{
				"fsRepoPath": path,
				"apiAddr":    cfg.Store.Options["url"],
			},
		}
		muxConfig = append(muxConfig, ipfsCfg)
	case "map":
		muxConfig = append(muxConfig, muxfs.MuxConfig{Type: "map"})
	case "mem":
		muxConfig = append(muxConfig, muxfs.MuxConfig{Type: "mem"})
	default:
		return nil, fmt.Errorf("unknown store type: %s", cfg.Store.Type)
	}

	if cfg.Repo.Type == "mem" {
		muxConfig = append(muxConfig, muxfs.MuxConfig{Type: "mem"})
	}

	return muxfs.New(ctx, muxConfig)
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
			return nil, fmt.Errorf("ipfs_http store requires 'url' option")
		}
		urlStr, ok := urli.(string)
		if !ok {
			return nil, fmt.Errorf("ipfs_http 'url' option must be a string")
		}
		return ipfs_http.NewFilesystem(map[string]interface{}{
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
