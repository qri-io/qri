package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/datapackage"
)

const (
	DATAPACKAGE_FILE_NAME = "datapackage.json"
	STATUS_FILE_NAME      = "status.json"
	REMOTES_FILE_NAME     = "remotes.json"
	REPO_DIR_NAME         = ".qri"
	CHANGES_DIR_NAME      = "changes"
	MIGRATIONS_DIR_NAME   = "migrations"
	COMMITS_DIR_NAME      = "commits"
)

// Repo represents a repository
type Repo struct {
	Package    *datapackage.DataPackage
	Status     *Status
	Commits    []*Commit
	Migrations []*Migration
	Changes    []*Change
}

// Initialize an empty repository
func (r *Repo) Init(dir string) error {
	// Create local .qri repository dir
	if err := os.Mkdir(filepath.Join(dir, REPO_DIR_NAME), 0666); err != nil {
		return errors.New(fmt.Sprintf("error creating .qri directory: %s", err.Error()))
	}

	// Create .qri/commits dir
	if err := ioutil.WriteFile(filepath.Join(dir, REPO_DIR_NAME, COMMITS_DIR_NAME), data, 0666); err != nil {
		return errors.New(fmt.Sprintf("error writing /%s/%s: %s", REPO_DIR_NAME, COMMITS_DIR_NAME, err.Error()))
	}

	// Create .qri/changes dir
	if err := os.Mkdir(filepath.Join(dir, REPO_DIR_NAME, CHANGES_DIR_NAME), 0666); err != nil {
		return errors.New(fmt.Sprintf("error creating .qri/changes: %s", err.Error()))
	}

	// Create local .qri/migrations dir
	if err := os.Mkdir(filepath.Join(dir, REPO_DIR_NAME, MIGRATIONS_DIR_NAME), 0666); err != nil {
		return errors.New(fmt.Sprintf("error creating .qri/migrations: %s", err.Error()))
	}

	if err := ioutil.WriteFile(filepath.Join(dir, REPO_DIR_NAME, STATUS_FILE_NAME), data, 0666); err != nil {
		return errors.New(fmt.Sprintf("error writing /%s/%s: %s", REPO_DIR_NAME, STATUS_FILE_NAME, err.Error()))
	}

	if err := ioutil.WriteFile(filepath.Join(dir, REPO_DIR_NAME, REMOTES_FILE_NAME), data, 0666); err != nil {
		return errors.New(fmt.Sprintf("error writing /%s/%s: %s", REPO_DIR_NAME, REMOTES_FILE_NAME, err.Error()))
	}

	return nil
}

// Clone grabs a dataset from a rmeote url & downloads it to a destination folder
func (r *Repo) Clone(dir string) error {
	return nil
}

// ReadLocal reads a repo from the local filesystem using a passed in directory
func (r *Repo) ReadLocal(dir string) error {
	// read package file
	if data, err := ioutil.ReadFile(filepath.Join(dir, DATAPACKAGE_FILE_NAME)); err == nil {
		if err := json.Unmarshal(data, r.Package); err != nil {
			return errors.New(fmt.Sprintf("error in %s: %s\n", DATAPACKAGE_FILE_NAME, err.Error()))
		}
	} else {
		return errors.New(fmt.Sprintf("Error reading %s: %s\n", DATAPACKAGE_FILE_NAME, err.Error()))
	}

	// make sure the repo directory exists
	if fi, err := os.Stat(filepath.Join(dir, REPO_DIR_NAME)); err != nil {
		return errors.New("this directory is not a recognized qri datapackage")
	} else if !fi.IsDir() {
		return errors.New("this directory is not a recognized qri datapackage")
	}

	// read status file
	if data, err := ioutil.ReadFile(filepath.Join(dir, REPO_DIR_NAME, STATUS_FILE_NAME)); err == nil {
		if err := json.Unmarshal(data, r.Package); err != nil {
			return errors.New(fmt.Sprintf("error in d%s: %s\n", DATAPACKAGE_FILE_NAME, err.Error()))
		}
	} else {
		return errors.New(fmt.Sprintf("error reading %s: %s\n", DATAPACKAGE_FILE_NAME, err.Error()))
	}

	return nil
}

// TODO - this should look for existing data in the given dir string
func (r *Repo) discoverData(dir string) ([]*datapackage.Resource, error) {
	return nil, nil
}

// Add to the repo
func (r *Repo) Add(dir string) error {
	if err := r.ReadLocal(dir); err != nil {
		return err
	}

	// for each resource:
	for _, resouce := range r.Package.Resources {
		bytes, err := resouce.FetchBytes()
		if err != nil {
			return err
		}

		// 1. check for migration changes, creating a migration file if needed

		// 2. check for data changes, creating change(s) if needed
	}

	return nil
}

// Commit looks at the current status of the directory, creates any necessary
// change & migration files for pushing
func (r *Repo) Commit(dir string) error {
	return nil
}

// Pull pulls from the 'origin' repo
func (r *Repo) Pull(dir string) error {
	return nil
}

// Push pushes to the 'origin' repo
func (r *Repo) Push(dir string) error {
	// 1. Check for changes to the data package, upload those if any
	// 2. Look through migration files for any file that doesn't have an id, if so push them up as api requests
	// 3. Look through change files for any file that doesn't have an id, if so, push them up as api requests
	return nil
}
