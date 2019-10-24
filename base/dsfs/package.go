package dsfs

import (
	"strings"

	"github.com/qri-io/qfs/cafs"
)

const (
	// transformScriptFilename is the name transform scripts will be written to
	transformScriptFilename = "transform_script"
	// vizsScriptFilename is the name transform scripts will be written to
	vizScriptFilename = "viz_script"
	// readmeScriptFilename is the name of the readme file that will be written to
	readmeScriptFilename = "readme_script"
)

// PackageFile specifies the different types of files that are
// stored in a package
type PackageFile int

const (
	// PackageFileUnknown is the default package file, which
	// should be erroneous, as there is no sensible default
	// for PackageFile
	PackageFileUnknown PackageFile = iota
	// PackageFileDataset is the main dataset.json file
	// that contains all dataset metadata, and is the only
	// required file to constitute a dataset
	PackageFileDataset
	// PackageFileStructure isolates this dataset's structure
	// in it's own file
	PackageFileStructure
	// PackageFileAbstract is the abstract verion of
	// structure
	PackageFileAbstract
	// PackageFileResources lists the resource datasets
	// that went into creating a dataset
	// TODO - I think this can be removed now that Transform exists
	PackageFileResources
	// PackageFileCommit isolates the user-entered
	// documentation of the changes to this dataset's history
	PackageFileCommit
	// PackageFileTransform isloates the concrete transform that
	// generated this dataset
	PackageFileTransform
	// PackageFileAbstractTransform is the abstract version of
	// the operation performed to create this dataset
	PackageFileAbstractTransform
	// PackageFileMeta encapsulates human-readable metadata
	PackageFileMeta
	// PackageFileViz isolates the data related to representing a dataset as a visualization
	PackageFileViz
	// PackageFileRenderedViz is the rendered visualization of the dataset
	PackageFileRenderedViz
	// PackageFileReadme is the raw readme of the dataset
	PackageFileReadme
	// PackageFileRenderedReadme is the rendered readme of the dataset
	PackageFileRenderedReadme
)

// filenames maps PackageFile to their filename counterparts
var filenames = map[PackageFile]string{
	PackageFileUnknown:           "",
	PackageFileDataset:           "dataset.json",
	PackageFileStructure:         "structure.json",
	PackageFileAbstract:          "abstract.json",
	PackageFileAbstractTransform: "abstract_transform.json",
	PackageFileResources:         "resources",
	PackageFileCommit:            "commit.json",
	PackageFileTransform:         "transform.json",
	PackageFileMeta:              "meta.json",
	PackageFileViz:               "viz.json",
	PackageFileRenderedViz:       "index.html",
	PackageFileReadme:            "readme.md",
	PackageFileRenderedReadme:    "readme.html",
}

// String implements the io.Stringer interface for PackageFile
func (p PackageFile) String() string {
	return p.Filename()
}

// Filename gives the canonical filename for a PackageFile
func (p PackageFile) Filename() string {
	return filenames[p]
}

// GetHashBase strips paths to return just the hash
func GetHashBase(in, network string) string {
	in = strings.TrimLeft(in, "/")
	in = strings.TrimPrefix(in, network)
	in = strings.TrimLeft(in, "/")
	return strings.Split(in, "/")[0]
}

// PackageFilepath returns the path to a package file for a given base path
// It relies relies on package storage conventions and cafs.Filestore path prefixes
// If you supply a path that does not match the filestore's naming conventions will
// return an invalid path
func PackageFilepath(store cafs.Filestore, path string, pf PackageFile) string {
	prefix := store.PathPrefix()
	if prefix == "" {
		return path
	}
	// Keep forward slashes in the path by using strings.Join instead of filepath.Join. This
	// will make IPFS happy on Windows, since it always wants "/" and not "\". The blank
	// path component in the front of this join ensures that the path begins with a "/" character.
	return strings.Join([]string{"", prefix, GetHashBase(path, prefix), pf.String()}, "/")
}
