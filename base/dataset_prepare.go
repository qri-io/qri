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
	"github.com/qri-io/qri/dsref"
	qerr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/profile"
)

// PrepareSaveRef works out a dataset reference for saving a dataset version.
// When a dataset exists, resolve the initID & path. When no dataset with that
// name exists, ensure a locally-unique dataset name  and create a new logbook
// history & InitID to write to. PrepareSaveRef returns a true boolean value
// if an initID was created
// successful calls to PrepareSaveRef always have an InitID, and will have the
// Path of the current version, if one exists
func PrepareSaveRef(
	ctx context.Context,
	pro *profile.Profile,
	book *logbook.Book,
	resolver dsref.Resolver,
	refStr string,
	bodyPathNameHint string,
	wantNewName bool,
) (dsref.Ref, bool, error) {
	log.Debugf("PrepareSaveRef refStr=%q bodyPathNameHint=%q wantNewName=%t", refStr, bodyPathNameHint, wantNewName)

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
		check := &dsref.Ref{Username: pro.Peername, Name: ref.Name}
		if _, resolveErr := resolver.ResolveRef(ctx, check); resolveErr == nil {
			if !wantNewName {
				// Name was inferred, and has previous version. Unclear if the user meant to create
				// a brand new dataset or if they wanted to add a new version to the existing dataset.
				// Raise an error recommending one of these course of actions.
				return ref, false, fmt.Errorf(`inferred dataset name already exists. To add a new commit to this dataset, run save again with the dataset reference %q. To create a new dataset, use --new flag`, check.Human())
			}
			ref.Name = GenerateAvailableName(ctx, pro, resolver, ref.Name)
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
	if ref.Username != "" && ref.Username != "me" && ref.Username != pro.Peername {
		return ref, false, fmt.Errorf("cannot save using a different username than %q", pro.Peername)
	}
	ref.Username = pro.Peername

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
			log.Error(badCaseErr)
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

	ref.InitID, err = book.WriteDatasetInit(ctx, ref.Name)
	log.Debugf("PrepareSaveRef created new initID=%q ref.Username=%q ref.Name=%q", ref.InitID, ref.Username, ref.Name)
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
	ds.Commit.Author = &dataset.User{ID: pro.ID.String()}

	// add any missing structure fields
	if err := detect.Structure(ds); err != nil && !errors.Is(err, dataset.ErrNoBody) {
		return err
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
		log.Debugf("ValidateDataset error=%q", err.Error())
		err = fmt.Errorf("invalid dataset: %s", err.Error())
		return
	}
	return nil
}
