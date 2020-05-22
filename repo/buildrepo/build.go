// Package buildrepo initializes a qri repo
package buildrepo

import (
	"sync"

	qipfs "github.com/qri-io/qfs/cafs/ipfs"
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

