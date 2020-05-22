// Package buildrepo initializes a qri repo
package buildrepo

import (
	"sync"

	qipfs "github.com/qri-io/qfs/cafs/ipfs"
<<<<<<< HEAD
	"github.com/qri-io/qfs/cafs/ipfs_http"
	"github.com/qri-io/qfs/httpfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/event/hook"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	fsrepo "github.com/qri-io/qri/repo/fs"
	"github.com/qri-io/qri/repo/profile"
=======
>>>>>>> d0915243... refactor(buildrepo): thin the file to only the necessary functions
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
		pluginLoadError = qipfs.LoadPlugins(path)
	}
	pluginLoadLock.Do(body)
	return pluginLoadError
}

