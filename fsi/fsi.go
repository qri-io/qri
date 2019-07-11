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
	"io/ioutil"
	"os"
	"sync"

	"github.com/qri-io/qri/repo"
)

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
	return nil, fmt.Errorf("TODO")
}

// Link connects a directory
func (fsi *FSI) Link(dirPath, refStr string) error {
	links, err := fsi.load()
	if err != nil {
		return err
	}

	l := &Link{Ref: refStr, Path: dirPath}
	links = append(links, l)

	return fsi.save(links)
}

// Unlink breaks the connection between a directory and a
func (fsi *FSI) Unlink(dirPath, ref string) error {

	return nil
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
