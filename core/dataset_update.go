package core

import (
	"fmt"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

type Commit struct {
	Author  *profile.Profile
	Prev    datastore.Key
	Changes *dataset.Dataset
	Data    cafs.SizeFile
	Message string
}

// Update adds a history entry updating a dataset
// TODO - work in progress
func (r *DatasetRequests) Update(commit *Commit, ref *repo.DatasetRef) error {
	store := r.repo.Store()
	ds := &dataset.Dataset{}

	prev, err := r.repo.GetDataset(commit.Prev)
	if err != nil {
		return fmt.Errorf("error getting previous dataset from repo: %s", err.Error())
	}

	// add all previous fields and any changes
	ds.Assign(prev, commit.Changes)

	// store file if one is provided
	if commit.Data != nil {
		size, err := commit.Data.Size()
		if err != nil {
			return fmt.Errorf("error getting data byte size: %s", err.Error())
		}
		path, err := store.Put(commit.Data, false)
		if err != nil {
			return fmt.Errorf("error putting data in store: %s", err.Error())
		}

		ds.Data = path
		ds.Length = int(size)
	}

	// TODO - validate dataset structure
	// structure may have been set by the metadata file above
	// by calling assign on ourselves with inferred structure in
	// the middle, any user-contributed schema metadata will overwrite
	// inferred metadata, but inferred schema properties will populate
	// empty fields
	// ds.Structure.Assign(ds.Structure, d.Structure)

	// TODO - there's a possibility that this'll come in as:
	// /ipfs/[hash]/dataset.json
	// is that what we want? or do we want the raw hash?
	ds.Previous = commit.Prev

	// TODO - should we be writing a "commit" file to the repository as well
	// that contains authorship & message information?

	// TODO - should this go into the save method?
	ds.Timestamp = time.Now().In(time.UTC)
	dspath, err := dsfs.SaveDataset(store, ds, true)
	if err != nil {
		return fmt.Errorf("error saving dataset: %s", err.Error())
	}

	name, err := r.repo.GetName(commit.Prev)
	if err != nil {
		return err
	}

	if err := r.repo.DeleteName(name); err != nil {
		return err
	}
	if err := r.repo.PutName(name, dspath); err != nil {
		return err
	}

	*ref = repo.DatasetRef{
		Name:    name,
		Path:    dspath,
		Dataset: ds,
	}

	return nil
}
