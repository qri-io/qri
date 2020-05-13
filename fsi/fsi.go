// Package fsi defines qri file system integration: representing a dataset as
// files in a directory on a user's computer. Using fsi, users can edit files
// as an interface for working with qri datasets.
//
// A dataset is "linked" to a directory through a `.qri_ref` dotfile that
// connects the folder to a version history stored in the local qri repository.
//
// files in a linked directory follow naming conventions that map to components
// of a dataset. eg: a file named "meta.json" in a linked directory maps to
// the dataset meta component. This mapping can be used to construct a dataset
// for read and write actions
package fsi

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/fsi/linkfile"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// package level logger
var (
	log = golog.Logger("fsi")

	// ErrNotLinkedToFilesystem is the err implementers should return when we
	// are expecting the dataset to have a file system link, but fsiPath is empty
	ErrNoLink = fmt.Errorf("dataset is not linked to the filesystem")
)

// GetLinkedFilesysRef returns whether a directory is linked to a dataset in your repo, and
// a reference to that dataset
func GetLinkedFilesysRef(dir string) (dsref.Ref, bool) {
	ref, err := linkfile.Read(filepath.Join(dir, linkfile.RefLinkHiddenFilename))
	return ref, err == nil
}

// RepoPath returns the standard path to an FSI file for a given file-system
// repo location
func RepoPath(repoPath string) string {
	return filepath.Join(repoPath, "fsi.qfb")
}

// FSI is a repo-side struct for coordinating file system integration
type FSI struct {
	// repository for resolving dataset names
	repo repo.Repo
	pub  event.Publisher
}

// NewFSI creates an FSI instance from a path to a links flatbuffer file
func NewFSI(r repo.Repo, pub event.Publisher) *FSI {
	if pub == nil {
		pub = &event.NilPublisher{}
	}
	return &FSI{repo: r, pub: pub}
}

// LinkedRefs returns a list of linked datasets and their connected directories
func (fsi *FSI) LinkedRefs(offset, limit int) ([]reporef.DatasetRef, error) {
	// TODO (b5) - figure out a better pagination / querying strategy here
	allRefs, err := fsi.repo.References(offset, 100000)
	if err != nil {
		return nil, err
	}

	var refs []reporef.DatasetRef
	for _, ref := range allRefs {
		if ref.FSIPath != "" {
			if offset > 0 {
				offset--
				continue
			}
			refs = append(refs, ref)
		}
		if len(refs) == limit {
			return refs, nil
		}
	}

	return refs, nil
}

// EnsureRefNotLinked checks if a ref already has an existing link on the file system
func (fsi *FSI) EnsureRefNotLinked(ref *reporef.DatasetRef) error {
	if stored, err := fsi.repo.GetRef(*ref); err == nil {
		// Check if there is already a link for this dataset, and if that link still exists.
		if stored.FSIPath != "" && linkfile.ExistsInDir(stored.FSIPath) {
			return fmt.Errorf("'%s' is already linked to %s", ref.AliasString(), stored.FSIPath)
		}
	}
	return nil
}

// CreateLink links a working directory to a dataset. Returns the reference alias, and a
// rollback function if no error occurs
func (fsi *FSI) CreateLink(dirPath, refStr string) (alias string, rollback func(), err error) {
	rollback = func() {}

	datasetRef, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return "", rollback, err
	}
	err = repo.CanonicalizeDatasetRef(fsi.repo, &datasetRef)
	if err != nil && err != repo.ErrNotFound && err != repo.ErrNoHistory {
		return datasetRef.String(), rollback, err
	}

	// todo(arqu): should utilize rollback as other operations bellow
	// can fail too
	if err := fsi.EnsureRefNotLinked(&datasetRef); err != nil {
		return "", rollback, err
	}

	// Link the FSIPath to the reference before putting it into the repo
	log.Debugf("fsi.CreateLink: linking ref=%q, FSIPath=%q", datasetRef, dirPath)
	datasetRef.FSIPath = dirPath
	if err = fsi.repo.PutRef(datasetRef); err != nil {
		return "", rollback, err
	}
	// If future steps fail, remove the ref we just put
	removeRefFunc := func() {
		log.Debugf("removing repo.ref %q during rollback", datasetRef)
		if err := fsi.repo.DeleteRef(datasetRef); err != nil {
			log.Debugf("error while removing repo.ref %q: %s", datasetRef, err)
		}
	}

	ref := reporef.ConvertToDsref(datasetRef)
	// Remove the path from the reference because linkfile's don't store full paths.
	ref.Path = ""
	linkFile, err := linkfile.WriteHiddenInDir(dirPath, ref)
	if err != nil {
		return "", removeRefFunc, err
	}
	// If future steps fail, remove the link file we just wrote to
	removeLinkAndRemoveRefFunc := func() {
		log.Debugf("removing linkFile %q during rollback", linkFile)
		if err := os.Remove(linkFile); err != nil {
			log.Debugf("error while removing linkFile %q: %s", linkFile, err)
		}
		removeRefFunc()
	}

	// Send an event to the bus about this checkout
	fsi.pub.Publish(event.ETFSICreateLinkEvent, event.FSICreateLinkEvent{
		FSIPath:  dirPath,
		Username: datasetRef.Peername,
		Dsname:   datasetRef.Name,
	})

	return datasetRef.AliasString(), removeLinkAndRemoveRefFunc, err
}

// ModifyLinkDirectory changes the FSIPath in the repo so that it is linked to the directory. Does
// not affect the .qri-ref linkfile in the working directory. Called when the command-line
// interface or filesystem watcher detects that a working folder has been moved.
// TODO(dlong): Add a filesystem watcher that behaves as described
// TODO(dlong): Perhaps add a `qri mv` command that explicitly changes a working directory location
func (fsi *FSI) ModifyLinkDirectory(dirPath, refStr string) error {
	ref, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil && err != repo.ErrNoHistory {
		return err
	}
	if ref.FSIPath == dirPath {
		return nil
	}

	log.Debugf("fsi.ModifyLinkDirectory: modify ref=%q, FSIPath was %q, changing to %q", ref, ref.FSIPath, dirPath)
	ref.FSIPath = dirPath
	return fsi.repo.PutRef(ref)
}

// ModifyLinkReference changes the reference that is in .qri-ref linkfile in the working directory.
// Does not affect the ref in the repo. Called when a rename command is invoked.
func (fsi *FSI) ModifyLinkReference(dirPath, refStr string) error {
	datasetRef, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(fsi.repo, &datasetRef); err != nil && err != repo.ErrNoHistory {
		return err
	}

	log.Debugf("fsi.ModifyLinkReference: modify linkfile at %q, ref=%q", dirPath, datasetRef)
	ref := reporef.ConvertToDsref(datasetRef)
	// Remove the path from the reference because linkfile's don't store full paths.
	ref.Path = ""
	if _, err = linkfile.WriteHiddenInDir(dirPath, ref); err != nil {
		return err
	}
	return nil
}

// Unlink removes the link file (.qri-ref) in the directory, and removes the fsi path
// from the reference in the refstore
func (fsi *FSI) Unlink(dirPath string, ref dsref.Ref) error {
	removeErr := os.Remove(filepath.Join(dirPath, linkfile.RefLinkHiddenFilename))
	if removeErr != nil {
		log.Debugf("removing link file: %s", removeErr.Error())
	}

	// Ref may be empty, which will mean only the link file should be removed
	if ref.IsEmpty() {
		return nil
	}

	// The FSIPath is *not* set, which removes it from the refstore
	datasetRef := reporef.RefFromDsref(ref)

	// if we're unlinking a ref without history, delete it
	if datasetRef.Path == "" {
		return fsi.repo.DeleteRef(datasetRef)
	}
	// Otherwise just update the refstore
	return fsi.repo.PutRef(datasetRef)
}

// Remove attempts to remove the dataset directory
func (fsi *FSI) Remove(dirPath string) error {
	// always attempt to remove the directory, ignoring "directory not empty" errors
	// os.Remove will fail if the directory isn't empty, which is the behaviour
	// we want
	if err := os.Remove(dirPath); err != nil && !strings.Contains(err.Error(), "directory not empty") {
		log.Errorf("removing directory: %s", err.Error())
	}
	return nil
}

// RemoveAll attempts to remove the dataset directory
// but also removes low value files first
func (fsi *FSI) RemoveAll(dirPath string) error {
	// clean up low value files before running Remove
	err := removeLowValueFiles(dirPath)
	if err != nil {
		log.Errorf("removing low value files: %s", err.Error())
	}
	if err := os.Remove(dirPath); err != nil && !strings.Contains(err.Error(), "directory not empty") {
		log.Errorf("removing directory: %s", err.Error())
	}
	return nil
}

func (fsi *FSI) getRepoRef(refStr string) (ref reporef.DatasetRef, err error) {
	ref, err = repo.ParseDatasetRef(refStr)
	if err != nil {
		return ref, err
	}

	if err = repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil {
		return ref, err
	}

	return fsi.repo.GetRef(ref)
}

func isLowValueFile(f os.FileInfo) bool {
	if f != nil {
		filesToBeRemoved := []string{
			// generic files
			"^\\..*\\.swp$", // Swap file for vim state

			// macOS specific files
			"^\\.DS_Store$",    // Stores custom folder attributes
			"^\\.AppleDouble$", // Stores additional file resources
			"^\\.LSOverride$",  // Contains the absolute path to the app to be used
			"^Icon\\r$",        // Custom Finder icon: http://superuser.com/questions/298785/icon-file-on-os-x-desktop
			"^\\._.*",          // Thumbnail
			"\\.Trashes",       // File that might appear on external disk
			"^__MACOSX$",       // Resource fork

			// Windows specific files
			"^Thumbs\\.db$",   // Image file cache
			"^ehthumbs\\.db$", // Folder config file
			"^Desktop\\.ini$", // Stores custom folder attributes
		}

		pattern := strings.Join(filesToBeRemoved, "|")
		matched, err := regexp.MatchString(pattern, filepath.Base(f.Name()))
		if err != nil {
			return false
		}
		if matched {
			return true
		}
	}
	return false
}

func getLowValueFiles(files *[]string) filepath.WalkFunc {
	return func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if isLowValueFile(f) {
			*files = append(*files, path)
		}
		return nil
	}
}

func removeLowValueFiles(dir string) error {
	var lowValueFiles []string
	err := filepath.Walk(dir, getLowValueFiles(&lowValueFiles))
	if err != nil {
		return err
	}
	for _, file := range lowValueFiles {
		if err = os.Remove(file); err != nil {
			return err
		}
	}
	return nil
}

// DeleteComponentFiles deletes all component files in the directory. Should only be used if
// removing an entire dataset, or if the dataset is about to be rewritten back to the filesystem.
func DeleteComponentFiles(dir string) error {
	dirComps, err := component.ListDirectoryComponents(dir)
	if err != nil {
		return err
	}
	for _, compName := range component.AllSubcomponentNames() {
		comp := dirComps.Base().GetSubcomponent(compName)
		if comp == nil {
			continue
		}
		err = os.Remove(comp.Base().SourceFile)
		if err != nil {
			log.Errorf("deleting file %q, error: %s", comp.Base().SourceFile, err)
			return err
		}
	}
	return nil
}
