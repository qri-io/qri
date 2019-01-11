package base

import (
	"bytes"
	"fmt"
	"io"

	datastore "github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/varName"
)

// PrepareDatasetSave prepares a set of changes for submission to SaveDataset
func PrepareDatasetSave(r repo.Repo, peername, name string) (prev, mutable *dataset.Dataset, body cafs.File, prevPath string, err error) {
	// Determine if the save is creating a new dataset or updating an existing dataset by
	// seeing if the name can canonicalize to a repo that we know about
	lookup := &repo.DatasetRef{Name: name, Peername: peername}
	if err = repo.CanonicalizeDatasetRef(r, lookup); err == repo.ErrNotFound {
		return &dataset.Dataset{}, &dataset.Dataset{}, nil, "", nil
	}

	prevPath = lookup.Path

	if prev, err = dsfs.LoadDataset(r.Store(), datastore.NewKey(prevPath)); err != nil {
		return
	}
	if prev.BodyPath != "" {
		body, err = dsfs.LoadBody(r.Store(), prev)
	}

	if mutable, err = dsfs.LoadDataset(r.Store(), datastore.NewKey(prevPath)); err != nil {
		return
	}

	// remove the Transform & previous commit
	mutable.Transform = nil
	mutable.Commit = nil
	return
}

// InferValues populates any missing fields that must exist to create a snapshot
func InferValues(pro *profile.Profile, name *string, ds *dataset.Dataset, body cafs.File) (res cafs.File, err error) {
	res = body
	// try to pick up a dataset name
	if *name == "" {
		*name = varName.CreateVarNameFromString(body.FileName())
	}

	// infer commit values
	if ds.Commit == nil {
		ds.Commit = &dataset.Commit{}
	}
	// NOTE: add author ProfileID here to keep the dataset package agnostic to
	// all identity stuff except keypair crypto
	ds.Commit.Author = &dataset.User{ID: pro.ID.String()}
	// TODO - infer title & message

	// if we don't have a structure, attempte to determine one
	if ds.Structure == nil && body != nil {
		// use a TeeReader that writes to a buffer to preserve data
		buf := &bytes.Buffer{}
		tr := io.TeeReader(body, buf)
		var df dataset.DataFormat

		df, err = detect.ExtensionDataFormat(body.FileName())
		if err != nil {
			log.Debug(err.Error())
			err = fmt.Errorf("invalid data format: %s", err.Error())
			return
		}

		ds.Structure, _, err = detect.FromReader(df, tr)
		if err != nil {
			log.Debug(err.Error())
			err = fmt.Errorf("determining dataset structure: %s", err.Error())
			return
		}
		// glue whatever we just read back onto the reader
		res = cafs.NewMemfileReader(body.FileName(), io.MultiReader(buf, body))
	}

	if ds.Transform != nil && ds.Transform.IsEmpty() {
		ds.Transform = nil
	}

	return
}

// ValidateDataset checks that a dataset is semantically valid
func ValidateDataset(name string, ds *dataset.Dataset) (err error) {
	if err = validate.ValidName(name); err != nil {
		return fmt.Errorf("invalid name: %s", err.Error())
	}

	// Ensure that dataset structure is valid
	if err = validate.Dataset(ds); err != nil {
		log.Debug(err.Error())
		err = fmt.Errorf("invalid dataset: %s", err.Error())
		return
	}

	return nil
}
