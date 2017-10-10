package fs_repo

import (
	"encoding/json"
	"github.com/ipfs/go-datastore"
	"io/ioutil"
	"os"
	"path/filepath"
)

type basepath string

func (bp basepath) filepath(f File) string {
	return filepath.Join(string(bp), Filepath(f))
}

func (bp basepath) readBytes(f File) ([]byte, error) {
	return ioutil.ReadFile(bp.filepath(f))
}

func (bp basepath) saveFile(d interface{}, f File) error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(bp.filepath(f), data, os.ModePerm)
}

// File represents a type file in a qri repository
type File int

const (
	// Unknown makes the default file value invalid
	FileUnknown File = iota
	// FileLockfile is the on-disk mutex lock
	FileLockfile
	// FileInfo stores information about this repository
	// like version number, size of repo, etc.
	FileInfo
	// FileProfile is this node's user profile
	FileProfile
	// FileConfig holds configuration specific to this repo
	FileConfig
	// FileDatasets holds the list of datasets
	FileDatasets
	//
	FileNamestore
	// FilePeers holds peer repositories
	// Ideally this won't stick around for long
	FilePeers
	// FileCache is the cache of datasets
	FileCache
	// FileAnalytics holds analytics data
	FileAnalytics
	// SearchIndex
	FileSearchIndex
)

var paths = map[File]string{
	FileUnknown:     "",
	FileLockfile:    "/repo.lock",
	FileInfo:        "/info.json",
	FileProfile:     "/profile.json",
	FileConfig:      "/config.json",
	FileDatasets:    "/datasets.json",
	FileNamestore:   "/namespace.json",
	FilePeers:       "/peers.json",
	FileCache:       "/cache.json",
	FileAnalytics:   "/analytics.json",
	FileSearchIndex: "/index.bleve",
}

// Filepath gives the relative filepath to a repofile
// in a given repository
func Filepath(rf File) string {
	return paths[rf]
}

// FileKey returns a datastore.Key reference for a
// given File
func FileKey(rf File) datastore.Key {
	return datastore.NewKey(Filepath(rf))
}
