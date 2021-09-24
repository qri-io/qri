package base

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	qerr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
)

// SaveSwitches is an alias for the switches that control how saves happen
type SaveSwitches = dsfs.SaveSwitches

// SaveDataset saves a version of the dataset for the given initID at the current path
func SaveDataset(
	ctx context.Context,
	r repo.Repo,
	writeDest qfs.Filesystem,
	author *profile.Profile,
	initID string,
	prevPath string,
	changes *dataset.Dataset,
	runState *run.State,
	sw SaveSwitches,
) (ds *dataset.Dataset, err error) {
	log.Debugw("SaveDataset", "initID", initID, "prevPath", prevPath, "author", author)
	if initID == "" {
		return nil, fmt.Errorf("SaveDataset requires an initID")
	}

	prev := &dataset.Dataset{}
	mutable := &dataset.Dataset{}
	fs := r.Filesystem()
	if prevPath != "" {
		// Load the dataset's most recent version, which will become the previous version after
		// this save operation completes.
		if prev, err = dsfs.LoadDataset(ctx, fs, prevPath); err != nil {
			return
		}
		if prev.BodyPath != "" {
			var body qfs.File
			body, err = dsfs.LoadBody(ctx, fs, prev)
			if err != nil {
				return nil, err
			}
			prev.SetBodyFile(body)
		}
		// Load a mutable copy of the dataset because most of the save path assuming we are doing
		// a patch update to the current head, and not a full replacement.
		if mutable, err = dsfs.LoadDataset(ctx, fs, prevPath); err != nil {
			return nil, err
		}

		// remove the commit. commit must be created from scratch with each new version
		mutable.Commit = nil
	}

	// Handle a change in structure format.
	if changes.BodyFile() != nil && prev.Structure != nil && changes.Structure != nil && prev.Structure.Format != changes.Structure.Format {
		log.Debugf("body formats differ. prev=%q new=%q", prev.Structure.Format, changes.Structure.Format)
		if sw.ConvertFormatToPrev {
			log.Debugf("changing structure format prev=%q new=%q", prev.Structure.Format, changes.Structure.Format)
			var f qfs.File
			f, err = ConvertBodyFormat(changes.BodyFile(), changes.Structure, prev.Structure)
			if err != nil {
				return nil, err
			}
			// Set the new format on the change structure.
			changes.Structure.Format = prev.Structure.Format
			changes.SetBodyFile(f)
		} else {
			err = fmt.Errorf("Refusing to change structure from %s to %s", prev.Structure.Format, changes.Structure.Format)
			return nil, err
		}
	}

	if !sw.Replace {
		// Treat the changes as a set of patches applied to the previous dataset
		mutable.Assign(changes)
		changes = mutable
	}

	// infer missing values
	if err = InferValues(author, changes); err != nil {
		return
	}

	// let's make history, if it exists
	changes.PreviousPath = prevPath

	// Write the dataset to storage and get back the new path
	ds, err = CreateDataset(ctx, r, writeDest, author, changes, prev, sw)
	if err != nil {
		return nil, err
	}
	ds.ID = initID

	// Write the save to logbook
	if err = r.Logbook().WriteVersionSave(ctx, author, ds, runState); err != nil {
		return nil, err
	}
	ds.ID = initID
	return ds, nil
}

// CreateDataset uses dsfs to add a dataset to a repo's store, updating the refstore
func CreateDataset(ctx context.Context, r repo.Repo, writeDest qfs.Filesystem, author *profile.Profile, ds, dsPrev *dataset.Dataset, sw SaveSwitches) (res *dataset.Dataset, err error) {
	log.Debugw("CreateDataset", "ds.ID", ds.ID)
	var path string

	// TODO(dustmop): Remove the dependence on the ds having an assigned Name. It is only
	// needed for updating the refstore. Either pass in the reference needed to update the refstore,
	// or move the refstore update out of this function.
	dsName := ds.Name
	if dsName == "" {
		return nil, fmt.Errorf("cannot create dataset without a name")
	}
	if err = Drop(ds, sw.Drop); err != nil {
		return nil, err
	}

	if err = validate.Dataset(ds); err != nil {
		log.Debugw("validate.Dataset", "err", err)
		return nil, fmt.Errorf("invalid dataset: %w", err)
	}

	if path, err = dsfs.CreateDataset(ctx, r.Filesystem(), writeDest, r.Bus(), ds, dsPrev, author.PrivKey, sw); err != nil {
		log.Debugf("dsfs.CreateDataset: %s", err)
		return nil, err
	}

	if ds.PreviousPath != "" && ds.PreviousPath != "/" {
		// should be ok to skip this error. we may not have the previous
		// reference locally
		repo.DeleteVersionInfoShim(ctx, r, dsref.Ref{
			ProfileID: author.ID.Encode(),
			Username:  author.Peername,
			Name:      dsName,
			Path:      ds.PreviousPath,
		})
	}

	log.Debugf("loading: %s", path)
	ds, err = dsfs.LoadDataset(ctx, r.Filesystem(), path)
	if err != nil {
		return nil, err
	}
	ds.ProfileID = author.ID.Encode()
	ds.Name = dsName
	ds.Peername = author.Peername
	ds.Path = path

	// TODO(dustmop): Reference is created here in order to update refstore. As we move to initID
	// and dscache, this will no longer be necessary, updating logbook will be enough.
	vi := dsref.ConvertDatasetToVersionInfo(ds)

	if err := repo.PutVersionInfoShim(ctx, r, &vi); err != nil {
		return nil, err
	}

	return ds, nil
}

// PrepareSaveRef works out a dataset reference for saving a dataset version.
// When a dataset exists, resolve the initID & path. When no dataset with that
// name exists, ensure a locally-unique dataset name  and create a new logbook
// history & InitID to write to. PrepareSaveRef returns a true boolean value
// if an initID was created
// successful calls to PrepareSaveRef always have an InitID, and will have the
// Path of the current version, if one exists
func PrepareSaveRef(
	ctx context.Context,
	author *profile.Profile,
	book *logbook.Book,
	resolver dsref.Resolver,
	refStr string,
	bodyPathNameHint string,
	wantNewName bool,
) (dsref.Ref, bool, error) {
	log.Debugw("PrepareSaveRef", "refStr", refStr, "bodyPathNameHint", bodyPathNameHint, "wantNeName", wantNewName)

	var badCaseErr error

	ref, err := dsref.ParseHumanFriendly(refStr)
	if errors.Is(err, dsref.ErrBadCaseName) {
		// save bad case error for later, will fail if dataset is new
		badCaseErr = err
	} else if errors.Is(err, dsref.ErrEmptyRef) {
		// User is calling save but didn't provide a dataset reference. Try to infer a usable name.
		if bodyPathNameHint == "" {
			bodyPathNameHint = "dataset"
		}
		basename := filepath.Base(bodyPathNameHint)
		basename = strings.TrimSuffix(basename, filepath.Ext(bodyPathNameHint))
		basename = strings.TrimSuffix(basename, ".")
		ref.Name = dsref.GenerateName(basename, "dataset_")

		// need to use profile username b/c resolver.ResolveRef can't handle "me"
		// shorthand
		check := &dsref.Ref{Username: author.Peername, Name: ref.Name}
		if _, resolveErr := resolver.ResolveRef(ctx, check); resolveErr == nil {
			if !wantNewName {
				// Name was inferred, and has previous version. Unclear if the user meant to create
				// a brand new dataset or if they wanted to add a new version to the existing dataset.
				// Raise an error recommending one of these course of actions.
				return ref, false, fmt.Errorf(`inferred dataset name already exists. To add a new commit to this dataset, run save again with the dataset reference %q. To create a new dataset, use --new flag`, check.Human())
			}
			ref.Name = GenerateAvailableName(ctx, author, resolver, ref.Name)
		}
	} else if errors.Is(err, dsref.ErrNotHumanFriendly) {
		return ref, false, err
	} else if err != nil {
		// If some other parse error happened, describe a valid dataset name.
		return ref, false, dsref.ErrDescribeValidName
	}

	// Validate that username is our own, it's not valid to try to save a dataset with someone
	// else's username. Without this check, base will replace the username with our own regardless,
	// it's better to have an error to display, rather than silently ignore it.
	if ref.Username != "" && ref.Username != "me" && ref.Username != author.Peername {
		return ref, false, fmt.Errorf("cannot save using a different username than %q", author.Peername)
	}
	ref.Username = author.Peername

	// attempt to resolve the reference
	if _, resolveErr := resolver.ResolveRef(ctx, &ref); resolveErr != nil {
		if !errors.Is(resolveErr, dsref.ErrRefNotFound) {
			return ref, false, resolveErr
		}
	} else if resolveErr == nil {
		if wantNewName {
			// Name was explicitly given, with the --new flag, but the name is already in use.
			// This is an error.
			return ref, false, qerr.New(ErrNameTaken, "dataset name has a previous version, cannot make new dataset")
		}

		if badCaseErr != nil {
			// name already exists but is a bad case, log a warning and then continue.
			log.Warn(badCaseErr)
		}

		// we have a valid previous reference & an initID, return!
		log.Debugw("PrepareSaveRef found previous initID", "initID", ref.InitID, "path", ref.Path)
		return ref, false, nil
	}

	// at this point we're attempting to create a new dataset.
	// If dataset name is using bad-case characters, and is not yet in use, fail with error.
	if badCaseErr != nil {
		return ref, true, badCaseErr
	}
	if !dsref.IsValidName(ref.Name) {
		return ref, true, fmt.Errorf("invalid dataset name: %s", ref.Name)
	}

	ref.InitID, err = book.WriteDatasetInit(ctx, author, ref.Name)
	log.Debugw("PrepareSaveRef complete", "ref", ref)
	return ref, true, err
}

// GenerateAvailableName creates a name for the dataset that is not currently in
// use. Generated names start with _2, implying the "_1" file is the original
// no-suffix name.
func GenerateAvailableName(ctx context.Context, pro *profile.Profile, resolver dsref.Resolver, prefix string) string {
	counter := 1
	for {
		counter++
		lookup := &dsref.Ref{Username: pro.Peername, Name: fmt.Sprintf("%s_%d", prefix, counter)}
		if _, err := resolver.ResolveRef(ctx, lookup); errors.Is(err, dsref.ErrRefNotFound) {
			return lookup.Name
		}
	}
}

// InferValues populates any missing fields that must exist to create a snapshot
func InferValues(pro *profile.Profile, ds *dataset.Dataset) error {
	// infer commit values
	if ds.Commit == nil {
		ds.Commit = &dataset.Commit{}
	}
	// NOTE: add author ProfileID here to keep the dataset package agnostic to
	// all identity stuff except keypair crypto
	ds.Commit.Author = &dataset.User{ID: pro.ID.Encode()}

	// add any missing structure fields
	if err := detect.Structure(ds); err != nil && !errors.Is(err, dataset.ErrNoBody) {
		return err
	}

	if ds.Transform != nil && ds.Transform.ScriptFile() == nil && ds.Transform.IsEmpty() {
		ds.Transform = nil
	}

	return nil
}
