package base

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// SaveSwitches is an alias for the switches that control how saves happen
type SaveSwitches = dsfs.SaveSwitches

// ErrNameTaken is an error for when a name for a new dataset is already being used
var ErrNameTaken = fmt.Errorf("name already in use")

// SaveDataset saves a version of the dataset for the given initID at the current path
func SaveDataset(ctx context.Context, r repo.Repo, writeDest qfs.Filesystem, initID, prevPath string, changes *dataset.Dataset, sw SaveSwitches) (ds *dataset.Dataset, err error) {
	log.Debugf("SaveDataset initID=%q prevPath=%q", initID, prevPath)
	var pro *profile.Profile
	if pro, err = r.Profile(); err != nil {
		return nil, err
	}
	if initID == "" {
		return nil, fmt.Errorf("SaveDataset requires an initID")
	}

	prev := &dataset.Dataset{}
	mutable := &dataset.Dataset{}
	if prevPath != "" {
		// Load the dataset's most recent version, which will become the previous version after
		// this save operation completes.
		if prev, err = dsfs.LoadDataset(ctx, r.Store(), prevPath); err != nil {
			return
		}
		if prev.BodyPath != "" {
			var body qfs.File
			body, err = dsfs.LoadBody(ctx, r.Store(), prev)
			if err != nil {
				return nil, err
			}
			prev.SetBodyFile(body)
		}
		// Load a mutable copy of the dataset because most of the save path assuming we are doing
		// a patch update to the current head, and not a full replacement.
		if mutable, err = dsfs.LoadDataset(ctx, r.Store(), prevPath); err != nil {
			return nil, err
		}

		// TODO(dustmop): Stop removing the transform once we move to apply, and untangle the
		// save command from applying a transform.
		// remove the Transform & commit
		// transform & commit must be created from scratch with each new version
		mutable.Transform = nil
		mutable.Commit = nil
	}

	// Save requires either a body or a structure.
	// TODO(dustmop): Saving with only a structure is currently broken. See TestSaveBasicCommands
	// in cmd/save_test.go
	if prevPath == "" && changes.BodyFile() == nil && changes.Structure == nil {
		return nil, fmt.Errorf("creating a new dataset requires a structure or a body")
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
			err = fmt.Errorf("Refusing to change structure from %s to %s",
				prev.Structure.Format, changes.Structure.Format)
			return nil, err
		}
	}

	if !sw.Replace {
		// Treat the changes as a set of patches applied to the previous dataset
		mutable.Assign(changes)
		changes = mutable
	}

	// infer missing values
	if err = InferValues(pro, changes); err != nil {
		return
	}

	// let's make history, if it exists
	changes.PreviousPath = prevPath

	// Write the dataset to storage and get back the new path
	ds, err = CreateDataset(ctx, r, writeDest, changes, prev, sw)
	if err != nil {
		return nil, err
	}

	// Write the save to logbook
	err = r.Logbook().WriteVersionSave(ctx, initID, changes)
	if err != nil && err != logbook.ErrNoLogbook {
		return ds, err
	}
	return ds, nil
}

// CreateDataset uses dsfs to add a dataset to a repo's store, updating the refstore
func CreateDataset(ctx context.Context, r repo.Repo, writeDest qfs.Filesystem, ds, dsPrev *dataset.Dataset, sw SaveSwitches) (res *dataset.Dataset, err error) {
	log.Debugf("CreateDataset ds=%#v dsPrev=%#v", ds, dsPrev)
	var (
		pro     *profile.Profile
		path    string
		resBody qfs.File
	)

	pro, err = r.Profile()
	if err != nil {
		log.Debugf("getting repo profile: %s", err)
		return
	}
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

	if err = ValidateDataset(ds); err != nil {
		log.Debugf("ValidateDataset error: %s", err)
		return
	}

	cafsWriteDest, ok := writeDest.(cafs.Filestore)
	if !ok {
		return nil, fmt.Errorf("write destination must be a cafs.Filstore")
	}

	if path, err = dsfs.CreateDataset(ctx, r.Store(), cafsWriteDest, ds, dsPrev, r.PrivateKey(), sw); err != nil {
		log.Debugf("dsfs.CreateDataset: %s", err)
		return nil, err
	}
	if ds.PreviousPath != "" && ds.PreviousPath != "/" {
		// should be ok to skip this error. we may not have the previous
		// reference locally
		repo.DeleteVersionInfoShim(r, dsref.Ref{
			ProfileID: pro.ID.String(),
			Username:  pro.Peername,
			Name:      dsName,
			Path:      ds.PreviousPath,
		})
	}

	ds, err = dsfs.LoadDataset(ctx, r.Store(), path)
	if err != nil {
		return nil, err
	}
	ds.ProfileID = pro.ID.String()
	ds.Name = dsName
	ds.Peername = pro.Peername
	ds.Path = path

	// TODO(dustmop): Reference is created here in order to update refstore. As we move to initID
	// and dscache, this will no longer be necessary, updating logbook will be enough.
	vi := dsref.ConvertDatasetToVersionInfo(ds)

	if err := repo.PutVersionInfoShim(r, &vi); err != nil {
		return nil, err
	}

	// need to open here b/c we might be doing a dry-run, which would mean we have
	// references to files in a store that won't exist after this function call
	// TODO (b5): this should be replaced with a call to OpenDataset with a qfs that
	// knows about the store
	if resBody, err = r.Store().Get(ctx, ds.BodyPath); err != nil {
		log.Error("error getting from store:", err.Error())
	}
	ds.SetBodyFile(resBody)
	return ds, nil
}
