package actions

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
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
func SaveDataset(ctx context.Context, node *p2p.QriNode, changes *dataset.Dataset, secrets map[string]string, scriptOut io.Writer, sw SaveDatasetSwitches) (ref repo.DatasetRef, err error) {
	var (
		prevPath string
		pro      *profile.Profile
		r        = node.Repo
	)

	prev, mutable, prevPath, err := base.PrepareDatasetSave(ctx, r, changes.Peername, changes.Name)
	if err != nil {
		return
	}

	if pro, err = r.Profile(); err != nil {
		return
	}

	if sw.DryRun {
		node.LocalStreams.PrintErr("🏃🏽‍♀️ dry run\n")
		// dry-runs store to an in-memory repo
		r, err = repo.NewMemRepo(pro, cafs.NewMapstore(), node.Repo.Filesystem(), profile.NewMemStore())
		if err != nil {
			return
		}
	}

	if changes.Transform != nil {
		// create a check func from a record of all the parts that the datasetPod is changing,
		// the startf package will use this function to ensure the same components aren't modified
		mutateCheck := mutatedComponentsFunc(changes)

		changes.Transform.Secrets = secrets
		if err = ExecTransform(ctx, node, changes, prev, scriptOut, mutateCheck); err != nil {
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
		if sw.ConvertFormatToPrev {
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

	if !sw.Replace {
		// Treat the changes as a set of patches applied to the previous dataset
		mutable.Assign(changes)
		changes = mutable
	}

	// infer missing values
	if err = base.InferValues(pro, changes); err != nil {
		return
	}

	// add a default viz if one is needed
	if sw.ShouldRender {
		base.MaybeAddDefaultViz(changes)
	}

	// let's make history, if it exists
	changes.PreviousPath = prevPath

	return base.CreateDataset(ctx, r, node.LocalStreams, changes, prev, sw.DryRun, sw.Pin, sw.Force, sw.ShouldRender)
}

// UpdateRemoteDataset brings a reference to the latest version, syncing to the
// latest history it can find over p2p & via any configured registry
func UpdateRemoteDataset(ctx context.Context, node *p2p.QriNode, ref *repo.DatasetRef, pin bool) (res repo.DatasetRef, err error) {
	var ldr base.LogDiffResult
	ldr, err = node.RequestLogDiff(ctx, ref)
	if err != nil {
		return
	}
	for _, add := range ldr.Add {
		if err = base.FetchDataset(ctx, node.Repo, &add, true, false); err != nil {
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
func AddDataset(ctx context.Context, node *p2p.QriNode, rc *remote.Client, remoteAddr string, ref *repo.DatasetRef) (err error) {
	log.Debugf("add dataset %s. remoteAddr: %s", ref.String(), remoteAddr)
	if !ref.Complete() {
		// TODO (ramfox): we should check to see if the dataset already exists locally
		// unfortunately, because of the nature of the ipfs filesystem commands, we don't
		// know if files we fetch are local only or possibly coming from the network.
		// instead, for now, let's just always try to add
		if _, err := ResolveDatasetRef(ctx, node, rc, remoteAddr, ref); err != nil {
			return err
		}
	}

	type addResponse struct {
		Ref   *repo.DatasetRef
		Error error
	}

	fetchCtx, cancelFetch := context.WithCancel(ctx)
	defer cancelFetch()
	responses := make(chan addResponse)
	tasks := 0

	if rc != nil && remoteAddr != "" {
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

			if err := rc.PullDataset(fetchCtx, ref, remoteAddr); err != nil {
				res.Error = err
				return
			}
			node.LocalStreams.PrintErr("🗼 fetched from registry\n")
			if pinner, ok := node.Repo.Store().(cafs.Pinner); ok {
				err := pinner.Pin(fetchCtx, ref.Path, true)
				res.Error = err
			}
		}(refCopy)
	}

	if node.Online {
		tasks++
		go func() {
			err := base.FetchDataset(fetchCtx, node.Repo, ref, true, true)
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
			cancelFetch()
			success = true
			*ref = *res.Ref
			break
		}
	}

	if !success {
		return fmt.Errorf("add failed: %s", err.Error())
	}

	prevRef, err := node.Repo.GetRef(repo.DatasetRef{Peername: ref.Peername, Name: ref.Name})
	if err != nil && err == repo.ErrNotFound {
		if err = node.Repo.PutRef(*ref); err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error putting dataset in repo: %s", err.Error())
		}
		return nil
	}
	if err != nil {
		return err
	}

	prevRef.Dataset, err = dsfs.LoadDataset(ctx, node.Repo.Store(), prevRef.Path)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error loading repo dataset: %s", prevRef.Path)
	}

	ref.Dataset, err = dsfs.LoadDataset(ctx, node.Repo.Store(), ref.Path)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error loading added dataset: %s", ref.Path)
	}

	return ReplaceRefIfMoreRecent(node, &prevRef, ref)
}

// ReplaceRefIfMoreRecent replaces the given ref in the ref store, if
// it is more recent then the ref currently in the refstore
func ReplaceRefIfMoreRecent(node *p2p.QriNode, prev, curr *repo.DatasetRef) error {
	var (
		prevTime time.Time
		currTime time.Time
	)
	if curr == nil || curr.Dataset == nil || curr.Dataset.Commit == nil {
		return fmt.Errorf("added dataset ref is not fully dereferenced")
	}
	currTime = curr.Dataset.Commit.Timestamp
	if prev == nil || prev.Dataset == nil || prev.Dataset.Commit == nil {
		return fmt.Errorf("previous dataset ref is not fully derefernced")
	}
	prevTime = prev.Dataset.Commit.Timestamp

	if prevTime.Before(currTime) || prevTime.Equal(currTime) {
		if err := node.Repo.PutRef(*curr); err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error putting dataset name in repo: %s", err.Error())
		}
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

		if err = r.Logbook().WriteNameAmend(context.TODO(), repo.ConvertToDsref(*current), new.Name); err != nil {
			return err
		}
	}

	if err = r.DeleteRef(*current); err != nil {
		return err
	}
	if err = r.PutRef(*new); err != nil {
		return err
	}

	return nil
}

// DeleteDataset removes a dataset from the store
func DeleteDataset(ctx context.Context, node *p2p.QriNode, ref *repo.DatasetRef) (err error) {
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
	log, err := base.DatasetLog(ctx, r, *ref, 10000, 0, false)
	if err != nil {
		return err
	}

	// for _, ref := range log {
	// 	time.Sleep(time.Millisecond * 50)
	// 	if err = base.UnpinDataset(r, ref); err != nil && err != repo.ErrNotPinner {
	// 		return err
	// 	}
	// }

	if err = r.DeleteRef(*ref); err != nil {
		return err
	}

	if err = base.UnpinDataset(ctx, r, *ref); err != nil && err != repo.ErrNotPinner {
		return err
	}

	err = r.Logbook().WriteVersionDelete(ctx, repo.ConvertToDsref(*ref), len(log))
	if err == logbook.ErrNotFound {
		return nil
	}

	return err
}
