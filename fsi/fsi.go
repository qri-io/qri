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
	"context"
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
)

var (
	// package level logger
	log = golog.Logger("fsi")
	// ErrNoLink is the err implementers should return when we are expecting the
	// dataset to have a file system link, but fsiPath is empty
	ErrNoLink = fmt.Errorf("dataset is not linked to the filesystem")
)

// FilesystemPathToLocal converts a qfs.Filesystem path that has an /fsi prefix
// to a local path
func FilesystemPathToLocal(qfsPath string) string {
	return strings.TrimPrefix(qfsPath, "/fsi")
}

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
		pub = event.NilBus
	}
	return &FSI{repo: r, pub: pub}
}

// ResolvedPath sets the Path value of a reference to the filesystem integration
// path if one exists, ignoring any prior Path value. If no FSI link exists
// ResolvedPath will return ErrNoLink
func (fsi *FSI) ResolvedPath(ref *dsref.Ref) error {
	if ref.InitID == "" {
		return fmt.Errorf("initID is required")
	}

	// TODO (b5) - currently causing tests to fail, we should be using dscache
	// if it exists
	// if dsc := fsi.repo.Dscache(); dsc != nil {
	// 	// TODO(b5) - dscache needs a lookup-by-id method
	// 	vi, err := dsc.LookupByName(*ref)
	// 	if err != nil {
	// 		return ErrNoLink
	// 	}
	// 	if vi.FSIPath != "" {
	// 		ref.Path = fmt.Sprintf("/fsi%s", vi.FSIPath)
	// 		return nil
	// 	}
	// 	return ErrNoLink
	// }

	// old method falls back to refstore
	vi, err := repo.GetVersionInfoShim(fsi.repo, *ref)
	if err != nil {
		return ErrNoLink
	}

	if vi.FSIPath != "" {
		ref.Path = fmt.Sprintf("/fsi%s", vi.FSIPath)
		return nil
	}
	return ErrNoLink
}

// IsFSIPath is a utility function that returns whether the given path is a
// local filesystem path
func IsFSIPath(path string) bool {
	return strings.HasPrefix(path, "/fsi")
}

// ListLinks returns a list of linked datasets and their connected
// directories
func (fsi *FSI) ListLinks(offset, limit int) ([]dsref.VersionInfo, error) {
	// TODO (b5) - figure out a better pagination / querying strategy here
	allRefs, err := repo.ListDatasetsShim(fsi.repo, offset, 100000)
	if err != nil {
		return nil, err
	}

	var refs []dsref.VersionInfo
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
func (fsi *FSI) EnsureRefNotLinked(ref dsref.Ref) error {
	if stored, err := repo.GetVersionInfoShim(fsi.repo, ref); err == nil {
		// Check if there is already a link for this dataset, and if that link still exists.
		if stored.FSIPath != "" && linkfile.ExistsInDir(stored.FSIPath) {
			return fmt.Errorf("%q is already linked to %s", ref.Human(), stored.FSIPath)
		}
	}
	return nil
}

// CreateLink links a working directory to an existing dataset. Returning
// updated VersionInfo and a rollback function if no error occurs
func (fsi *FSI) CreateLink(dirPath string, ref dsref.Ref) (vi *dsref.VersionInfo, rollback func(), err error) {
	ctx := context.TODO()
	rollback = func() {}

	// todo(arqu): should utilize rollback as other operations bellow
	// can fail too
	if err := fsi.EnsureRefNotLinked(ref); err != nil {
		return nil, rollback, err
	}

	// Link the FSIPath to the reference before putting it into the repo
	log.Debugf("fsi.CreateLink: linking ref=%q, FSIPath=%q", ref, dirPath)
	vi, err = fsi.ModifyLinkDirectory(dirPath, ref)
	if err != nil {
		return nil, rollback, err
	}

	// If future steps fail, remove the ref we just put
	removeRefFunc := func() {
		log.Debugf("removing repo.ref %q during rollback", ref)
		if _, err := repo.DeleteVersionInfoShim(fsi.repo, ref); err != nil {
			log.Debugf("error while removing repo.ref %q: %s", ref, err)
		}
	}

	linkfilePath, err := fsi.ModifyLinkReference(dirPath, ref)
	if err != nil {
		return nil, removeRefFunc, err
	}

	// If future steps fail, remove the link file we just wrote to
	removeLinkAndRemoveRefFunc := func() {
		log.Debugf("removing linkFile %q during rollback", linkfilePath)
		if err := os.Remove(linkfilePath); err != nil {
			log.Debugf("error while removing linkFile %q: %s", linkfilePath, err)
		}
		removeRefFunc()
	}

	// Send an event to the bus about this checkout
	err = fsi.pub.Publish(ctx, event.ETFSICreateLinkEvent, event.FSICreateLinkEvent{
		FSIPath:  dirPath,
		Username: vi.Username,
		Dsname:   vi.Name,
	})
	if err != nil {
		return nil, removeLinkAndRemoveRefFunc, err
	}

	err = fsi.pub.Publish(ctx, event.ETDatasetCreateLink, event.DsChange{
		InitID:     ref.InitID, // versionInfo probably coming from old Refstore
		Username:   vi.Username,
		PrettyName: vi.Name,
		Dir:        dirPath,
	})
	if err != nil {
		return nil, removeLinkAndRemoveRefFunc, err
	}

	return vi, removeLinkAndRemoveRefFunc, err
}

// ModifyLinkDirectory changes the FSIPath in the repo so that it is linked to the directory. Does
// not affect the .qri-ref linkfile in the working directory. Called when the command-line
// interface or filesystem watcher detects that a working folder has been moved.
// TODO(dlong): Add a filesystem watcher that behaves as described
// TODO(dlong): Perhaps add a `qri mv` command that explicitly changes a working directory location
func (fsi *FSI) ModifyLinkDirectory(dirPath string, ref dsref.Ref) (*dsref.VersionInfo, error) {
	vi, err := repo.GetVersionInfoShim(fsi.repo, ref)
	if err != nil {
		return nil, err
	}
	if vi.FSIPath == dirPath {
		return vi, nil
	}

	log.Debugf("fsi.ModifyLinkDirectory: modify ref=%q, FSIPath was %q, changing to %q", ref, vi.FSIPath, dirPath)
	vi.FSIPath = dirPath
	err = repo.PutVersionInfoShim(fsi.repo, vi)
	return vi, err
}

// ModifyLinkReference changes the reference that is in .qri-ref linkfile in the working directory.
// Does not affect the ref in the repo. Called when a rename command is invoked.
func (fsi *FSI) ModifyLinkReference(dirPath string, ref dsref.Ref) (string, error) {
	log.Debugf("fsi.ModifyLinkReference: modify linkfile at %q, ref=%q", dirPath, ref)
	// Remove the path from the reference because linkfile's don't store full paths.
	ref.Path = ""
	return linkfile.WriteHiddenInDir(dirPath, ref)
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

	vi, err := repo.GetVersionInfoShim(fsi.repo, ref)
	if err != nil {
		return err
	}
	vi.FSIPath = ""

	// if we're unlinking a ref without history, delete it
	if ref.Path == "" {
		_, err := repo.DeleteVersionInfoShim(fsi.repo, ref)
		return err
	}
	// Otherwise just update the refstore
	return repo.PutVersionInfoShim(fsi.repo, vi)
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
