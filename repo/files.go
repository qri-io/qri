package repo

import (
	"github.com/ipfs/go-datastore"
)

// File represents a type file in a qri repository
type File int

const (
	// Unknown makes the default file value invalid
	FileUnknown File = iota
	// FileInfo stores information about this repository
	// like version number, size of repo, etc.
	FileInfo
	// FileProfile is this node's user profile
	FileProfile
	// FileConfig holds configuration specific to this repo
	FileConfig
	// FileNamespace holds this repo's local namespace
	FileNamespace
	// FileQueryResults holds resource hashes for a query hash key
	FileQueryResults
	// FileResourceMeta holds metadata hashes for a resource key
	FileResourceMeta
	// FileResourceQueries holds query hashes for a resource key
	FileResourceQueries
	// FilePeerRepos holds peer repositories
	// Ideally this won't stick around for long
	FilePeerRepos
)

var paths = map[File]string{
	FileUnknown:         "",
	FileInfo:            "/info",
	FileProfile:         "/profile",
	FileConfig:          "/config",
	FileNamespace:       "/namespace",
	FileQueryResults:    "/query_results",
	FileResourceMeta:    "/resource_meta",
	FileResourceQueries: "/resource_queries",
	FilePeerRepos:       "/peer_repos",
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
