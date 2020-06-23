// Package buildrepo initializes a qri repo
package buildrepo

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/event/hook"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	fsrepo "github.com/qri-io/qri/repo/fs"
	"github.com/qri-io/qri/repo/profile"
)

// Options provides additional fields to new
type Options struct {
	Filesystem *muxfs.Mux
	Logbook    *logbook.Book
	Dscache    *dscache.Dscache
}

// New is the canonical method for building a repo
func New(ctx context.Context, path string, cfg *config.Config, opts ...func(o *Options)) (repo.Repo, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	var err error
	if o.Filesystem == nil {
		if o.Filesystem, err = NewFilesystem(ctx, cfg); err != nil {
			return nil, err
		}
	}

	pro, err := profile.NewProfile(cfg.Profile)
	if err != nil {
		return nil, err
	}

	switch cfg.Repo.Type {
	case "fs":
		if o.Logbook == nil {
			if o.Logbook, err = newLogbook(o.Filesystem, pro, path); err != nil {
				return nil, err
			}
		}
		if o.Dscache == nil {
			if o.Dscache, err = newDscache(ctx, o.Filesystem, o.Logbook, pro.Peername, path); err != nil {
				return nil, err
			}
		}

		r, err := fsrepo.NewRepo(path, o.Filesystem, o.Logbook, o.Dscache, pro)
		return r, err
	case "mem":
		return repo.NewMemRepo(ctx, pro, o.Filesystem)
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

// TODO (b5) - if we had a better logbook constructor, this wouldn't need to exist
func newLogbook(fs qfs.Filesystem, pro *profile.Profile, repoPath string) (book *logbook.Book, err error) {
	logbookPath := filepath.Join(repoPath, "logbook.qfb")
	return logbook.NewJournal(pro.PrivKey, pro.Peername, fs, logbookPath)
}

func newDscache(ctx context.Context, fs qfs.Filesystem, book *logbook.Book, username, repoPath string) (*dscache.Dscache, error) {
	// This seems to be a bug, the repoPath does not end in "qri" in some tests.
	if !strings.HasSuffix(repoPath, "qri") {
		return nil, fmt.Errorf("invalid repo path: %q", repoPath)
	}
	dscachePath := filepath.Join(repoPath, "dscache.qfb")
	return dscache.NewDscache(ctx, fs, []hook.ChangeNotifier{book}, username, dscachePath), nil
}
