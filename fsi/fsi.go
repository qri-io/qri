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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo"
)

// QriRefFilename is the name of the file that links a folder to a dataset.
// The file contains a dataset reference that declares the link
// ref files are the authoritative definition of weather a folder is linked
// or not
const QriRefFilename = ".qri-ref"

// GetLinkedFilesysRef returns whether a directory is linked to a
// dataset in your repo, and the reference to that dataset.
func GetLinkedFilesysRef(dir string) (string, bool) {
	data, err := ioutil.ReadFile(filepath.Join(dir, QriRefFilename))
	if err == nil {
		return strings.TrimSpace(string(data)), true
	}
	return "", false
}

// RepoPath returns the standard path to an FSI file for a given file-system
// repo location
func RepoPath(repoPath string) string {
	return filepath.Join(repoPath, "fsi.qfb")
}

// FSI is a repo-side struct for coordinating file system integration
type FSI struct {
	// path to qri repo links filepath
	linksPath string
	// read/write lock
	lock sync.Mutex
	// repository for resolving dataset names
	repo repo.Repo
}

// NewFSI creates an FSI instance from a path to a links flatbuffer file
func NewFSI(r repo.Repo) *FSI {
	return &FSI{repo: r}
}

// LinkedRefs returns a list of linked datasets and their connected directories
func (fsi *FSI) LinkedRefs(offset, limit int) ([]repo.DatasetRef, error) {
	// TODO (b5) - figure out a better pagination / querying strategy here
	allRefs, err := fsi.repo.References(offset, 100000)
	if err != nil {
		return nil, err
	}

	var refs []repo.DatasetRef
	skipped := 0
	for _, ref := range allRefs {
		if ref.FSIPath != "" {
			if skipped > offset {
				skipped++
			} else {
				refs = append(refs, ref)
			}
		}
		if len(refs) == limit {
			return refs, nil
		}
	}

	return refs, nil
}

// CreateLink connects a directory
func (fsi *FSI) CreateLink(dirPath, refStr string) (string, error) {
	ref, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return "", err
	}

	if err = repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil && err != repo.ErrNotFound {
		return ref.String(), err
	}

	if stored, err := fsi.repo.GetRef(ref); err == nil {
		if stored.FSIPath != "" {
			// There is already a link for this dataset, see if that link still exists.
			targetPath := filepath.Join(stored.FSIPath, QriRefFilename)
			if _, err := os.Stat(targetPath); err == nil {
				return "", fmt.Errorf("'%s' is already linked to %s", ref.AliasString(), stored.FSIPath)
			}
		}
	}

	ref.FSIPath = dirPath
	err = fsi.repo.PutRef(ref)

	if err = writeLinkFile(dirPath, ref.AliasString()); err != nil {
		return "", err
	}

	return ref.AliasString(), err
}

// UpdateLink changes an existing link entry
func (fsi *FSI) UpdateLink(dirPath, refStr string) (string, error) {
	ref, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return "", err
	}

	if err = repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil {
		return ref.String(), err
	}

	ref.FSIPath = dirPath
	err = fsi.repo.PutRef(ref)
	return ref.String(), err
}

// Unlink breaks the connection between a directory and a dataset
func (fsi *FSI) Unlink(dirPath, refStr string) error {
	ref, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return err
	}

	if err = repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil {
		return err
	}

	ref.FSIPath = ""
	return fsi.repo.PutRef(ref)
}

// WriteComponents writes components of the dataset to the given path, as individual files.
func WriteComponents(ds *dataset.Dataset, dirPath string) error {
	// Get individual meta and schema components.
	meta := ds.Meta
	ds.Meta = nil
	schema := ds.Structure.Schema
	ds.Structure.Schema = nil

	// Body format to use later.
	bodyFormat := ds.Structure.Format

	// Structure is kept in the dataset.
	ds.Structure.Format = ""
	ds.Structure.Qri = ""

	// Commit, viz, transform are never written as individual files.
	ds.Commit = nil
	ds.Viz = nil
	ds.Transform = nil

	// Meta component.
	if meta != nil {
		meta.DropDerivedValues()
		if !meta.IsEmpty() {
			data, err := json.MarshalIndent(meta, "", " ")
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(filepath.Join(dirPath, "meta.json"), data, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}

	// Schema component.
	if len(schema) > 0 {
		data, err := json.MarshalIndent(schema, "", " ")
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dirPath, "schema.json"), data, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Body component.
	bf := ds.BodyFile()
	if bf != nil {
		data, err := ioutil.ReadAll(bf)
		if err != nil {
			return err
		}
		ds.BodyPath = ""
		var bodyFilename string
		switch bodyFormat {
		case "csv":
			bodyFilename = "body.csv"
		case "json":
			bodyFilename = "body.json"
		default:
			return fmt.Errorf("unknown body format: %s", bodyFormat)
		}
		err = ioutil.WriteFile(filepath.Join(dirPath, bodyFilename), data, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Dataset (everything else).
	ds.DropDerivedValues()
	// TODO(dlong): Should more of these move to DropDerivedValues?
	ds.Qri = ""
	ds.Name = ""
	ds.Peername = ""
	ds.PreviousPath = ""
	if ds.Structure != nil && ds.Structure.IsEmpty() {
		ds.Structure = nil
	}
	if !ds.IsEmpty() {
		data, err := json.MarshalIndent(ds, "", " ")
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dirPath, "dataset.json"), data, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fsi *FSI) getRepoRef(refStr string) (ref repo.DatasetRef, err error) {
	ref, err = repo.ParseDatasetRef(refStr)
	if err != nil {
		return ref, err
	}

	if err = repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil {
		return ref, err
	}

	return fsi.repo.GetRef(ref)
}

func writeLinkFile(dir, linkstr string) error {
	dir = filepath.Join(dir, QriRefFilename)
	return ioutil.WriteFile(dir, []byte(linkstr), os.ModePerm)
}

func removeLinkFile(dir string) error {
	dir = filepath.Join(dir, QriRefFilename)
	return os.Remove(dir)
}
