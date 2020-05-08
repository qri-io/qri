package base

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// PrepareHeadDatasetVersion prepares to save by loading the head commit, opening the body file,
// and constructing a mutable version that has no transform or commit.
func PrepareHeadDatasetVersion(ctx context.Context, r repo.Repo, peername, name string) (curr, mutable *dataset.Dataset, currPath string, err error) {
	// TODO(dustmop): We should not be calling CanonicalizeDatasetRef here. It's already been
	// called up in lib, which means that we've thrown information away. Furthermore, we
	// should be relying on stable identifiers this low down the stack. Instead, load the dataset
	// head (if it exists) by using the initID, and pass it into base.Save.

	// Determine if the save is creating a new dataset or updating an existing dataset by
	// seeing if the name can canonicalize to a repo that we know about
	lookup := &reporef.DatasetRef{Name: name, Peername: peername}
	if err = repo.CanonicalizeDatasetRef(r, lookup); err == repo.ErrNotFound || lookup.Path == "" {
		return &dataset.Dataset{}, &dataset.Dataset{}, "", nil
	}

	currPath = lookup.Path
	log.Debugf("loading currPath: %s. lookup result: %v", currPath, lookup)

	if curr, err = dsfs.LoadDataset(ctx, r.Store(), currPath); err != nil {
		return
	}
	if curr.BodyPath != "" {
		var body qfs.File
		body, err = dsfs.LoadBody(ctx, r.Store(), curr)
		if err != nil {
			return
		}
		curr.SetBodyFile(body)
	}

	// Load a mutable copy of the dataset because most of the save path assuming we are doing
	// a patch update to the current head, and not a full replacement.
	if mutable, err = dsfs.LoadDataset(ctx, r.Store(), currPath); err != nil {
		return
	}

	// TODO(dustmop): Stop removing the transform once we move to apply, and untangle the
	// save command from applying a transform.
	// remove the Transform & commit
	// transform & commit must be created from scratch with each new version
	mutable.Transform = nil
	mutable.Commit = nil
	return
}

// MaybeInferName infer a name for the dataset if none is set
func MaybeInferName(ds *dataset.Dataset) string {
	if ds.Name == "" {
		filename := ds.BodyFile().FileName()
		basename := strings.TrimSuffix(filename, filepath.Ext(filename))
		return dsref.GenerateName(basename, "dataset_")
	}
	return ""
}

// InferValues populates any missing fields that must exist to create a snapshot
func InferValues(pro *profile.Profile, ds *dataset.Dataset) error {
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

	return nil
}

// ValidateDataset checks that a dataset is semantically valid
func ValidateDataset(ds *dataset.Dataset) (err error) {
	// Ensure that dataset structure is valid
	if err = validate.Dataset(ds); err != nil {
		log.Debug(err.Error())
		err = fmt.Errorf("invalid dataset: %s", err.Error())
		return
	}

	return nil
}
