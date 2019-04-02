package actions

import (
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// SaveDataset initializes a dataset from a dataset pointer and data file
func SaveDataset(node *p2p.QriNode, changes *dataset.Dataset, secrets map[string]string, scriptOut io.Writer, dryRun, pin, convertFormatToPrev, force, shouldRender bool) (ref repo.DatasetRef, err error) {
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
		node.LocalStreams.PrintErr("🏃🏽‍♀️ dry run\n")
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
			return
		}
		// changes.Transform.SetScriptFile(mutable.Transform.ScriptFile())
		node.LocalStreams.PrintErr("✅ transform complete\n")
	}

	if prevPath == "" && changes.BodyFile() == nil && changes.Structure == nil {
		err = fmt.Errorf("creating a new dataset requires a structure or a body")
		return
	}

	if changes.BodyFile() != nil && prev.Structure != nil && changes.Structure != nil && prev.Structure.Format != changes.Structure.Format {
		if convertFormatToPrev {
			var f qfs.File
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

	// infer missing values, adding a default viz if shouldRender is true
	if err = base.InferValues(pro, changes, shouldRender); err != nil {
		return
	}

	// let's make history, if it exists
	changes.PreviousPath = prevPath

	return base.CreateDataset(r, node.LocalStreams, changes, prev, dryRun, pin, force, shouldRender)
}

// UpdateRemoteDataset brings a reference to the latest version, syncing to the
// latest history it can find over p2p & via any configured registry
func UpdateRemoteDataset(node *p2p.QriNode, ref *repo.DatasetRef, pin bool) (res repo.DatasetRef, err error) {
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
			node.LocalStreams.PrintErr("🗼 fetched from registry\n")
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
		node.LocalStreams.PrintErr("📝 listing dataset for p2p discovery\n")
	} else {
		node.LocalStreams.PrintErr("unlisting dataset from p2p discovery\n")
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
