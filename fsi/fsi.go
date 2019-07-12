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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"fmt"

	"github.com/qri-io/qri/repo"
)

// QriRefFilename links the current working folder to a dataset by containing a ref to it.
const QriRefFilename = ".qri-ref"

// GetLinkedFilesysRef returns whether the current directory is linked to a
// dataset in your repo, and the reference to that dataset.
func GetLinkedFilesysRef(dir string) (string, bool) {
	data, err := ioutil.ReadFile(QriRefFilename)
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
func NewFSI(r repo.Repo, path string) *FSI {
	return &FSI{linksPath: path, repo: r}
}

// Links returns a list of linked datasets and their connected directories
func (fsi *FSI) Links() ([]*Link, error) {
	return fsi.load()
}

// CreateLink connects a directory
func (fsi *FSI) CreateLink(dirPath, refStr string) (string, error) {
	links, err := fsi.load()
	if err != nil {
		return "", err
	}

	ref, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return "", err
	}

	if err = repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil {
		return ref.String(), err
	}

	alias := ref.AliasString()

	for _, l := range links {
		if l.Alias == alias {
			err = fmt.Errorf("'%s' is already linked to %s", alias, l.Path)
			return "", err
		}
	}

	l := &Link{Path: dirPath, Ref: ref.String(), Alias: ref.AliasString() }
	links = append(links, l)

	if err = writeLinkFile(dirPath, ref.String()); err != nil {
		return "", err
	}

	return ref.String(), fsi.save(links)
}

// UpdateLink changes an existing link entry
func (fsi *FSI) UpdateLink(dirPath, refStr string) (string, error) {
	links, err := fsi.load()
	if err != nil {
		return "", err
	}

	ref, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return "", err
	}

	if err = repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil {
		return ref.String(), err
	}

	alias := ref.AliasString()

	for i, l := range links {
		if l.Alias == alias {
			l := &Link{Path: dirPath, Ref: ref.String(), Alias: ref.AliasString() }
			links[i] = l
			return ref.String(), fsi.save(links)
		}
	}

	if err = writeLinkFile(dirPath, ref.String()); err != nil {
		return "", err
	}

	return "", fmt.Errorf("%s is not linked", ref)
}

// Unlink breaks the connection between a directory and a
func (fsi *FSI) Unlink(dirPath, refStr string) (string, error) {
	links, err := fsi.load()
	if err != nil {
		return "", err
	}

	ref, err := repo.ParseDatasetRef(refStr)
	if err != nil {
		return "", err
	}

	if err = repo.CanonicalizeDatasetRef(fsi.repo, &ref); err != nil {
		return ref.String(), err
	}

	alias := ref.AliasString()

	for i, l := range links {
		if l.Alias == alias {
			links = links.Remove(i)
			
			if err = removeLinkFile(dirPath); err != nil {
				return "", err
			}

			return "", fsi.save(links)
		}
	}


	return "", fmt.Errorf("%s is not linked", ref)
}

func (fsi *FSI) load() (links, error) {
	fsi.lock.Lock()
	defer fsi.lock.Unlock()

	data, err := ioutil.ReadFile(fsi.linksPath)
	if err != nil {
		return nil, err
	}

	return unmarshalLinksFlatbuffer(data)
}

func (fsi *FSI) save(ls links) error {
	fsi.lock.Lock()
	defer fsi.lock.Unlock()

	data := ls.FlatbufferBytes()
	return ioutil.WriteFile(fsi.linksPath, data, os.ModePerm)
}

func writeLinkFile(dir, linkstr string) error {
	dir = filepath.Join(dir, QriRefFilename)
	return ioutil.WriteFile(dir, []byte(linkstr), os.ModePerm)
}

func removeLinkFile(dir string) error {
	dir = filepath.Join(dir, QriRefFilename)
	return os.Remove(dir)
}