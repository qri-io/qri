package fsrepo

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
		log.Debug(err.Error())
		return err
	}
	return ioutil.WriteFile(bp.filepath(f), data, os.ModePerm)
}

// File represents a type file in a qri repository
type File int

const (
	// FileUnknown makes the default file value invalid
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
	// FileQueryLogs is a log of all queries in order they're run
	FileQueryLogs
	// FileRefstore is a file for the user's local namespace
	FileRefstore
	// FileRefCache stores known references to datasets
	FileRefCache
	// FilePeers holds peer repositories
	// Ideally this won't stick around for long
	FilePeers
	// FileCache is the cache of datasets
	FileCache
	// FileAnalytics holds analytics data
	FileAnalytics
	// FileSearchIndex is the path to a search index
	FileSearchIndex
	// FileChangeRequests is a file of change requests
	FileChangeRequests
)

var paths = map[File]string{
	FileUnknown:        "",
	FileLockfile:       "/repo.lock",
	FileInfo:           "/info.json",
	FileProfile:        "/profile.json",
	FileConfig:         "/config.json",
	FileDatasets:       "/datasets.json",
	FileQueryLogs:      "/queries.json",
	FileRefstore:       "/namespace.json",
	FileRefCache:       "/ref_cache.json",
	FilePeers:          "/peers.json",
	FileCache:          "/cache.json",
	FileAnalytics:      "/analytics.json",
	FileSearchIndex:    "/index.bleve",
	FileChangeRequests: "/change_requests.json",
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
