package actions

import (
	"fmt"
	"io"

	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/fs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// OpenDataset prepares a dataset for use, checking each component
// for populated Path or Byte suffixed fields, consuming those fields to
// set File handlers that are ready for reading
func OpenDataset(fsys fs.Filesystem, ds *dataset.Dataset) (err error) {
	if ds.BodyFile() == nil {
		if err = ds.OpenBodyFile(fsys); err != nil {
			return
		}
	}
	if ds.Transform != nil && ds.Transform.ScriptFile() == nil {
		if err = ds.Transform.OpenScriptFile(fsys); err != nil {
			return
		}
	}
	if ds.Viz != nil && ds.Viz.ScriptFile() == nil {
		if err = ds.Viz.OpenScriptFile(fsys); err != nil {
			return
		}
	}
	return
}

// CloseDataset ensures all open dataset files are closed
func CloseDataset(ds *dataset.Dataset) (err error) {
	if ds.BodyFile() != nil {
		if err = ds.BodyFile().Close(); err != nil {
			return
		}
	}
	if ds.Transform != nil && ds.Transform.ScriptFile() != nil {
		if err = ds.Transform.ScriptFile().Close(); err != nil {
			return
		}
	}
	if ds.Viz != nil && ds.Viz.ScriptFile() != nil {
		if err = ds.Viz.ScriptFile().Close(); err != nil {
			return
		}
	}

	return
}

// SaveDataset initializes a dataset from a dataset pointer and data file
func SaveDataset(node *p2p.QriNode, changes *dataset.Dataset, secrets map[string]string, scriptOut io.Writer, dryRun, pin, convertFormatToPrev bool) (ref repo.DatasetRef, err error) {
	var (
		prevPath string
		pro      *profile.Profile
		r        = node.Repo
	)

	prev, mutable, prevPath, err := base.PrepareDatasetSave(r, changes.Peername, changes.Name)
	if err != nil {
		return
	}

	if pro, err = r.Profile(); err != nil {
		return
	}

	if dryRun {
		node.LocalStreams.Print("🏃🏽‍♀️ dry run\n")
		// dry-runs store to an in-memory repo
		r, err = repo.NewMemRepo(pro, cafs.NewMapstore(), node.Repo.Filesystem(), profile.NewMemStore(), nil)
		if err != nil {
			return
		}
	}

	if changes.Transform != nil {
		// create a check func from a record of all the parts that the datasetPod is changing,
		// the startf package will use this function to ensure the same components aren't modified
		mutateCheck := mutatedComponentsFunc(changes)

		changes.Transform.Secrets = secrets
		if err = ExecTransform(node, changes, scriptOut, mutateCheck); err != nil {
			log.Error(err)
			return
		}
		// changes.Transform.SetScriptFile(mutable.Transform.ScriptFile())
		node.LocalStreams.Print("✅ transform complete\n")
	}

	// Infer any values about the incoming change before merging it with the previous version.
	if err = base.InferValues(pro, changes); err != nil {
		return
	}

	if changes.BodyFile() != nil && prev.Structure != nil && changes.Structure != nil && prev.Structure.Format != changes.Structure.Format {
		if convertFormatToPrev {
			var f fs.File
			f, err = base.ConvertBodyFormat(changes.BodyFile(), changes.Structure, prev.Structure)
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

	// apply the changes to the previous dataset.
	mutable.Assign(changes)
	changes = mutable

	// let's make history, if it exists:
	changes.PreviousPath = prevPath

	return base.CreateDataset(r, node.LocalStreams, changes, prev, dryRun, pin)
}

// UpdateDataset brings a reference to the latest version, syncing over p2p if the reference is
// in a peer's namespace, re-running a transform if the reference is owned by this profile
func UpdateDataset(node *p2p.QriNode, ref *repo.DatasetRef, secrets map[string]string, scriptOut io.Writer, dryRun, pin bool) (res repo.DatasetRef, err error) {
	if dryRun {
		node.LocalStreams.Print("🏃🏽‍♀️ dry run\n")
	}

	if err = repo.CanonicalizeDatasetRef(node.Repo, ref); err == repo.ErrNotFound {
		err = fmt.Errorf("unknown dataset '%s'. please add before updating", ref.AliasString())
		return
	} else if err != nil {
		return
	}

	if !base.InLocalNamespace(node.Repo, ref) {
		var ldr base.LogDiffResult
		ldr, err = node.RequestLogDiff(ref)
		if err != nil {
			return
		}
		for _, add := range ldr.Add {
			if err = base.FetchDataset(node.Repo, &add, true, false); err != nil {
				return
			}
		}
		if err = node.Repo.PutRef(ldr.Head); err != nil {
			return
		}
		res = ldr.Head
		// TODO - currently we're not loading the body here
		return
	}

	return localUpdate(node, ref, secrets, scriptOut, dryRun, pin)
}

// localUpdate runs a transform on a local dataset and returns the new dataset ref and body
func localUpdate(node *p2p.QriNode, ref *repo.DatasetRef, secrets map[string]string, scriptOut io.Writer, dryRun, pin bool) (res repo.DatasetRef, err error) {
	ds := ref.Dataset

	if ds == nil {
		if err = base.ReadDataset(node.Repo, ref); err != nil {
			log.Error(err)
			return
		}
		ds = ref.Dataset
	}

	if ds.Transform == nil {
		err = fmt.Errorf("transform script is required to automate updates to your own datasets")
		return
	}

	ds.Name = ref.Name
	ds.Transform.Secrets = secrets
	if err = OpenDataset(node.Repo.Filesystem(), ds); err != nil {
		return
	}

	prevRef := &repo.DatasetRef{
		Peername:  ref.Peername,
		Name:      ref.Name,
		Path:      ref.Path,
		ProfileID: ref.ProfileID,
	}
	if err = base.ReadDataset(node.Repo, prevRef); err != nil {
		log.Error(err)
		return
	}
	prev := prevRef.Dataset
	if err = OpenDataset(node.Repo.Filesystem(), prev); err != nil {
		return
	}

	node.LocalStreams.Print("🤖 executing transform\n")

	err = ExecTransform(node, ds, scriptOut, nil)
	if err != nil {
		log.Error(err)
		return
	}
	node.LocalStreams.Print("✅ transform complete\n")
	ds.PreviousPath = ref.Path

	return base.CreateDataset(node.Repo, node.LocalStreams, ds, prev, dryRun, pin)
}

// AddDataset fetches & pins a dataset to the store, adding it to the list of stored refs
func AddDataset(node *p2p.QriNode, ref *repo.DatasetRef) (err error) {
	if !ref.Complete() {
		if local, err := ResolveDatasetRef(node, ref); err != nil {
			return err
		} else if local {
			return fmt.Errorf("error: dataset %s already exists in repo", ref)
		}
	}

	type addResponse struct {
		Ref   *repo.DatasetRef
		Error error
	}

	responses := make(chan addResponse)
	tasks := 0

	rc := node.Repo.Registry()
	if rc != nil {
		tasks++

		refCopy := &repo.DatasetRef{
			Peername:  ref.Peername,
			ProfileID: ref.ProfileID,
			Name:      ref.Name,
			Path:      ref.Path,
		}

		go func(ref *repo.DatasetRef) {
			res := addResponse{Ref: ref}

			// always send on responses channel
			defer func() {
				responses <- res
			}()

			ng, err := newNodeGetter(node)
			if err != nil {
				res.Error = err
				return
			}

			capi, err := newIPFSCoreAPI(node)
			if res.Error != nil {
				res.Error = err
				return
			}

			if err := rc.DsyncFetch(node.Context(), ref.Path, ng, capi.Block()); err != nil {
				res.Error = err
				return
			}
			node.LocalStreams.Print("🗼 fetched from registry\n")
			if pinner, ok := node.Repo.Store().(cafs.Pinner); ok {
				err := pinner.Pin(ref.Path, true)
				res.Error = err
			}
		}(refCopy)
	}

	if node.Online {
		tasks++
		go func() {
			err := base.FetchDataset(node.Repo, ref, true, true)
			responses <- addResponse{
				Ref:   ref,
				Error: err,
			}
		}()
	}

	if tasks == 0 {
		return fmt.Errorf("no registry configured and node is not online")
	}

	success := false
	for i := 0; i < tasks; i++ {
		res := <-responses
		err = res.Error
		if err == nil {
			success = true
			*ref = *res.Ref
			break
		}
	}

	if !success {
		return fmt.Errorf("add failed: %s", err.Error())
	}

	if err = node.Repo.PutRef(*ref); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error putting dataset name in repo: %s", err.Error())
	}

	return nil
}

// SetPublishStatus configures the publish status of a stored reference
func SetPublishStatus(node *p2p.QriNode, ref *repo.DatasetRef, published bool) (err error) {
	if published {
		node.LocalStreams.Print("📝 listing dataset for p2p discovery\n")
	} else {
		node.LocalStreams.Print("unlisting dataset from p2p discovery\n")
	}
	return base.SetPublishStatus(node.Repo, ref, published)
}

// ModifyDataset alters a reference by changing what dataset it refers to
func ModifyDataset(node *p2p.QriNode, current, new *repo.DatasetRef, isRename bool) (err error) {
	r := node.Repo
	if err := validate.ValidName(new.Name); err != nil {
		return err
	}
	if err := repo.CanonicalizeDatasetRef(r, current); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error with existing reference: %s", err.Error())
	}
	err = repo.CanonicalizeDatasetRef(r, new)
	if err == nil {
		if isRename {
			return fmt.Errorf("dataset '%s/%s' already exists", new.Peername, new.Name)
		}
	} else if err != repo.ErrNotFound {
		log.Debug(err.Error())
		return fmt.Errorf("error with new reference: %s", err.Error())
	}
	if isRename {
		new.Path = current.Path
	}

	if err = r.DeleteRef(*current); err != nil {
		return err
	}
	if err = r.PutRef(*new); err != nil {
		return err
	}

	return r.LogEvent(repo.ETDsRenamed, *new)
}

// DeleteDataset removes a dataset from the store
func DeleteDataset(node *p2p.QriNode, ref *repo.DatasetRef) (err error) {
	r := node.Repo

	if err = repo.CanonicalizeDatasetRef(r, ref); err != nil {
		log.Debug(err.Error())
		return err
	}

	p, err := r.GetRef(*ref)
	if err != nil {
		log.Debug(err.Error())
		return err
	}
	if ref.Path != p.Path {
		return fmt.Errorf("given path does not equal most recent dataset path: cannot delete a specific save, can only delete entire dataset history. use `me/dataset_name` to delete entire dataset")
	}

	// TODO - this is causing bad things in our tests. For some reason core repo explodes with nil
	// references when this is on and go test ./... is run from $GOPATH/github.com/qri-io/qri
	// let's upgrade IPFS to the latest version & try again
	// log, err := base.DatasetLog(r, *ref, 10000, 0, false)
	// if err != nil {
	// 	return err
	// }

	// for _, ref := range log {
	// 	time.Sleep(time.Millisecond * 50)
	// 	if err = base.UnpinDataset(r, ref); err != nil {
	// 		return err
	// 	}
	// }

	if err = r.DeleteRef(*ref); err != nil {
		return err
	}

	if err = base.UnpinDataset(r, *ref); err != nil && err != repo.ErrNotPinner {
		return err
	}

	return r.LogEvent(repo.ETDsDeleted, *ref)
}
