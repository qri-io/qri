package base

import (
	"context"
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/startf"
)

// SaveDatasetSwitches provides toggleable flags to SaveDataset that control
// save behaviour
type SaveDatasetSwitches struct {
	Replace             bool
	DryRun              bool
	Pin                 bool
	ConvertFormatToPrev bool
	Force               bool
	ShouldRender        bool
}

// SaveDataset initializes a dataset from a dataset pointer and data file
func SaveDataset(ctx context.Context, r repo.Repo, str ioes.IOStreams, changes *dataset.Dataset, secrets map[string]string, scriptOut io.Writer, sw SaveDatasetSwitches) (ref repo.DatasetRef, err error) {
	var (
		prevPath string
		pro      *profile.Profile
	)

	prev, mutable, prevPath, err := PrepareDatasetSave(ctx, r, changes.Peername, changes.Name)
	if err != nil {
		return
	}

	if pro, err = r.Profile(); err != nil {
		return
	}

	if sw.DryRun {
		str.PrintErr("🏃🏽‍♀️ dry run\n")

		// dry-runs store to an in-memory repo
		r, err = repo.NewMemRepo(pro, cafs.NewMapstore(), r.Filesystem(), profile.NewMemStore())
		if err != nil {
			return
		}
	}

	if changes.Transform != nil {
		// create a check func from a record of all the parts that the datasetPod is changing,
		// the startf package will use this function to ensure the same components aren't modified
		mutateCheck := startf.MutatedComponentsFunc(changes)

		opts := []func(*startf.ExecOpts){
			startf.AddQriRepo(r),
			startf.AddMutateFieldCheck(mutateCheck),
			startf.SetOutWriter(scriptOut),
			startf.SetSecrets(secrets),
		}

		if err = startf.ExecScript(ctx, changes, prev, opts...); err != nil {
			return
		}

		str.PrintErr("✅ transform complete\n")
	}

	if prevPath == "" && changes.BodyFile() == nil && changes.Structure == nil {
		err = fmt.Errorf("creating a new dataset requires a structure or a body")
		return
	}

	if changes.BodyFile() != nil && prev.Structure != nil && changes.Structure != nil && prev.Structure.Format != changes.Structure.Format {
		if sw.ConvertFormatToPrev {
			var f qfs.File
			f, err = ConvertBodyFormat(changes.BodyFile(), changes.Structure, prev.Structure)
			if err != nil {
				return
			}
			// Set the new format on the change structure.
			changes.Structure.Format = prev.Structure.Format
			changes.SetBodyFile(f)
		} else {
			err = fmt.Errorf("Refusing to change structure from %s to %s",
				prev.Structure.Format, changes.Structure.Format)
			return
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

	// TODO(dlong): Remove this, stop generating a default viz.
	// add a default viz if one is needed
	if sw.ShouldRender {
		MaybeAddDefaultViz(changes)
	}

	// let's make history, if it exists
	changes.PreviousPath = prevPath

	return CreateDataset(ctx, r, str, changes, prev, sw.DryRun, sw.Pin, sw.Force, sw.ShouldRender)
}

// CreateDataset uses dsfs to add a dataset to a repo's store, updating all
// references within the repo if successful
func CreateDataset(ctx context.Context, r repo.Repo, streams ioes.IOStreams, ds, dsPrev *dataset.Dataset, dryRun, pin, force, shouldRender bool) (ref repo.DatasetRef, err error) {
	var (
		pro     *profile.Profile
		path    string
		resBody qfs.File
	)

	pro, err = r.Profile()
	if err != nil {
		return
	}

	if err = ValidateDataset(ds); err != nil {
		return
	}

	if path, err = dsfs.CreateDataset(ctx, r.Store(), ds, dsPrev, r.PrivateKey(), pin, force, shouldRender); err != nil {
		return
	}
	if ds.PreviousPath != "" && ds.PreviousPath != "/" {
		prev := repo.DatasetRef{
			ProfileID: pro.ID,
			Peername:  pro.Peername,
			Name:      ds.Name,
			Path:      ds.PreviousPath,
		}

		// should be ok to skip this error. we may not have the previous
		// reference locally
		_ = r.DeleteRef(prev)
	}
	ref = repo.DatasetRef{
		ProfileID: pro.ID,
		Peername:  pro.Peername,
		Name:      ds.Name,
		Path:      path,
	}

	if err = r.PutRef(ref); err != nil {
		return
	}

	// TODO (b5): confirm these assignments happen in dsfs.CreateDataset with tests
	ds.ProfileID = pro.ID.String()
	ds.Peername = pro.Peername
	ds.Path = path
	if err = r.Logbook().WriteVersionSave(ctx, ds); err != nil && err != logbook.ErrNoLogbook {
		return
	}

	if err = ReadDataset(ctx, r, &ref); err != nil {
		return
	}

	// need to open here b/c we might be doing a dry-run, which would mean we have
	// references to files in a store that won't exist after this function call
	// TODO (b5): this should be replaced with a call to OpenDataset with a qfs that
	// knows about the store
	if resBody, err = r.Store().Get(ctx, ref.Dataset.BodyPath); err != nil {
		log.Error("error getting from store:", err.Error())
	}
	ref.Dataset.SetBodyFile(resBody)
	return
}
