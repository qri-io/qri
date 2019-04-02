package base

import (
	"bytes"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/varName"
)

// PrepareDatasetSave prepares a set of changes for submission to SaveDataset
// prev is the previous dataset, if it exists
// body is the previous dataset body, if it exists
// mutable is the previous dataset, but without the commit and transform, making it
// sutable for mutation/combination with any potential changes requested by the user
// we do not error if the dataset is not found in the repo, instead we return all
// empty values
// TODO (b5): input parameters here assume the store can properly resolve the previous dataset path
// through canonicalization (looking the name up in the repo). The value given by the input dataset
// document may differ, and we should probably respect that value if it does
func PrepareDatasetSave(r repo.Repo, peername, name string) (prev, mutable *dataset.Dataset, prevPath string, err error) {
	// Determine if the save is creating a new dataset or updating an existing dataset by
	// seeing if the name can canonicalize to a repo that we know about
	lookup := &repo.DatasetRef{Name: name, Peername: peername}
	if err = repo.CanonicalizeDatasetRef(r, lookup); err == repo.ErrNotFound {
		return &dataset.Dataset{}, &dataset.Dataset{}, "", nil
	}

	prevPath = lookup.Path

	if prev, err = dsfs.LoadDataset(r.Store(), prevPath); err != nil {
		return
	}
	if prev.BodyPath != "" {
		var body qfs.File
		body, err = dsfs.LoadBody(r.Store(), prev)
		if err != nil {
			return
		}
		prev.SetBodyFile(body)
	}

	if mutable, err = dsfs.LoadDataset(r.Store(), prevPath); err != nil {
		return
	}

	// remove the Transform & previous commit
	// transform & commit must be created from scratch with each new version
	mutable.Transform = nil
	mutable.Commit = nil
	return
}

// InferValues populates any missing fields that must exist to create a snapshot
func InferValues(pro *profile.Profile, ds *dataset.Dataset, inferViz bool) error {
	// try to pick up a dataset name
	if ds.Name == "" {
		ds.Name = varName.CreateVarNameFromString(ds.BodyFile().FileName())
	}

	// infer commit values
	if ds.Commit == nil {
		ds.Commit = &dataset.Commit{}
	}
	// NOTE: add author ProfileID here to keep the dataset package agnostic to
	// all identity stuff except keypair crypto
	ds.Commit.Author = &dataset.User{ID: pro.ID.String()}
	// TODO - infer title & message

	// if we don't have a structure or schema then attempt to determine one
	body := ds.BodyFile()
	if body != nil && (ds.Structure == nil || ds.Structure.Schema == nil) {
		// use a TeeReader that writes to a buffer to preserve data
		buf := &bytes.Buffer{}
		tr := io.TeeReader(body, buf)
		var df dataset.DataFormat

		df, err := detect.ExtensionDataFormat(body.FileName())
		if err != nil {
			log.Debug(err.Error())
			err = fmt.Errorf("invalid data format: %s", err.Error())
			return err
		}

		guessedStructure, _, err := detect.FromReader(df, tr)
		if err != nil {
			log.Debug(err.Error())
			err = fmt.Errorf("determining dataset structure: %s", err.Error())
			return err
		}

		// attach the structure, schema, and formatConfig, as appropriate
		if ds.Structure == nil {
			ds.Structure = guessedStructure
		}
		if ds.Structure.Schema == nil {
			ds.Structure.Schema = guessedStructure.Schema
		}
		if ds.Structure.FormatConfig == nil {
			ds.Structure.FormatConfig = guessedStructure.FormatConfig
		}

		// glue whatever we just read back onto the reader
		// TODO (b5)- this may ruin readers that transparently depend on a read-closer
		// we should consider a method on qfs.File that allows this non-destructive read pattern
		ds.SetBodyFile(qfs.NewMemfileReader(body.FileName(), io.MultiReader(buf, body)))
	}

	if ds.Transform != nil && ds.Transform.ScriptFile() == nil && ds.Transform.IsEmpty() {
		ds.Transform = nil
	}

	if inferViz && ds.Viz == nil {
		AddDefaultViz(ds)
	}

	return nil
}

// ValidateDataset checks that a dataset is semantically valid
func ValidateDataset(ds *dataset.Dataset) (err error) {
	if err = validate.ValidName(ds.Name); err != nil {
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
