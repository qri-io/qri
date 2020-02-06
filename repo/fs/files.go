package fsrepo

import (
	"encoding/json"
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
	// FileConfig holds configuration specific to this repo
	FileConfig
	// FileDatasets holds the list of datasets
	FileDatasets
	// FileEventLogs is a log of all queries in order they're run
	FileEventLogs
	// FileJSONRefs is a file for the user's local namespace
	// No longer in use
	FileJSONRefs
	// FileDscache is a flatbuffer file of this repo's dataset cache
	FileDscache
	// FileRefs is a flatbuffer file of this repo's dataset references
	FileRefs
	// FilePeers holds peer repositories
	// Ideally this won't stick around for long
	FilePeers
	// FileAnalytics holds analytics data
	FileAnalytics
	// FileSearchIndex is the path to a search index
	FileSearchIndex
	// FileSelectedRefs is the path to the current ref selection
	FileSelectedRefs
	// FileChangeRequests is a file of change requests
	FileChangeRequests
)

var paths = map[File]string{
	FileUnknown:        "",
	FileLockfile:       "/repo.lock",
	FileInfo:           "/info.json",
	FileConfig:         "/config.json",
	FileDatasets:       "/datasets.json",
	FileEventLogs:      "/events.json",
	FileJSONRefs:       "/ds_refs.json",
	FileDscache:        "/dscache.fbs",
	FileRefs:           "/refs.fbs",
	FilePeers:          "/peers.json",
	FileAnalytics:      "/analytics.json",
	FileSearchIndex:    "/index.bleve",
	FileSelectedRefs:   "/selected_refs.json",
	FileChangeRequests: "/change_requests.json",
}

// Filepath gives the relative filepath to a repofiles
// in a given repository
func Filepath(rf File) string {
	return paths[rf]
}
