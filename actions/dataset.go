package actions

import (
	"fmt"
	"os"
	"strings"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// SaveDataset initializes a dataset from a dataset pointer and data file
func SaveDataset(node *p2p.QriNode, changesPod *dataset.DatasetPod, secrets map[string]string, dryRun, pin, convertFormatToPrev bool) (ref repo.DatasetRef, body cafs.File, err error) {
	var (
		prev                     *dataset.Dataset
		prevPath                 string
		bodyFile, changeBodyFile cafs.File
		pro                      *profile.Profile
		changes                  = &dataset.Dataset{}
		r                        = node.Repo
	)

	// set ds to dataset head, or empty dataset if no history
	prev, bodyFile, prevPath, err = base.PrepareDatasetSave(r, changesPod.Peername, changesPod.Name)
	if err != nil {
		return
	}

	pro, err = r.Profile()
	if err != nil {
		return
	}

	if dryRun {
		node.LocalStreams.Print("🏃🏽‍♀️ dry run\n")
		// dry-runs store to an in-memory repo
		r, err = repo.NewMemRepo(pro, cafs.NewMapstore(), profile.NewMemStore(), nil)
		if err != nil {
			return
		}
	}

	if changeBodyFile, err = base.DatasetPodBodyFile(node.Repo.Store(), changesPod); err != nil {
		return
	}
	if err = changes.Decode(changesPod); err != nil {
		return
	}

	if changes.Transform != nil {
		// create a check func from a record of all the parts that the datasetPod is changing,
		// the startf package will use this function to ensure the same components aren't modified
		mutateCheck := mutatedComponentsFunc(changesPod)
		if changes.Transform.Script == nil {
			if strings.HasPrefix(changes.Transform.ScriptPath, "/ipfs") || strings.HasPrefix(changes.Transform.ScriptPath, "/map") || strings.HasPrefix(changes.Transform.ScriptPath, "/cafs") {
				var f cafs.File
				f, err = node.Repo.Store().Get(datastore.NewKey(changes.Transform.ScriptPath))
				if err != nil {
					return
				}
				changes.Transform.Script = f
			} else {
				var f *os.File
				f, err = os.Open(changes.Transform.ScriptPath)
				if err != nil {
					return
				}
				changes.Transform.Script = f
			}
		}
		// TODO - consider making this a standard method on dataset.Transform
		script := cafs.NewMemfileReader(changes.Transform.ScriptPath, changes.Transform.Script)

		var config map[string]interface{}
		if changesPod.Transform == nil {
			config = make(map[string]interface{})
		} else {
			config = changesPod.Transform.Config
		}
		bodyFile, err = ExecTransform(node, prev, script, bodyFile, secrets, config, mutateCheck)
		if err != nil {
			return
		}
		node.LocalStreams.Print("✅ transform complete\n")
	}

	// Infer any values about the incoming change before merging it with the previous version.
	changeBodyFile, err = base.InferValues(pro, &changesPod.Name, changes, changeBodyFile)
	if err != nil {
		return
	}

	if prev.Structure != nil && changes.Structure != nil && prev.Structure.Format != changes.Structure.Format {
		if convertFormatToPrev {
			changeBodyFile, err = base.ConvertBodyFormat(changeBodyFile, changes.Structure,
				prev.Structure)
			if err != nil {
				return
			}
			// Set the new format on the change structure.
			changes.Structure.Format = prev.Structure.Format
		} else {
			err = fmt.Errorf("Refusing to change structure from %s to %s",
				prev.Structure.Format, changes.Structure.Format)
			return
		}
	}

	// apply changes to the previous path, set changes to the result
	prev.Assign(changes)
	changes = prev
	clearPaths(changes)

	if changeBodyFile != nil {
		changes.BodyPath = ""
		bodyFile = changeBodyFile
	}

	// let's make history, if it exists:
	changes.PreviousPath = prevPath
	return base.CreateDataset(r, node.LocalStreams, changesPod.Name, changes, bodyFile, dryRun, pin)
}

// for now it's very important we remove any path references before saving
// we should remove this in the long run, but not without extensive tests in
// dsfs, and dsdiff packages, both of which are very sensitive to paths being present
func clearPaths(ds *dataset.Dataset) {
	if ds.Meta != nil {
		ds.Meta.SetPath("")
	}
	if ds.Structure != nil {
		ds.Structure.SetPath("")
	}
	if ds.Viz != nil {
		ds.Viz.SetPath("")
	}
	if ds.Transform != nil {
		ds.Transform.SetPath("")
	}
}

// UpdateDataset brings a reference to the latest version, syncing over p2p if the reference is
// in a peer's namespace, re-running a transform if the reference is owned by this profile
func UpdateDataset(node *p2p.QriNode, ref *repo.DatasetRef, secrets map[string]string, dryRun, pin bool) (res repo.DatasetRef, body cafs.File, err error) {
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

	return localUpdate(node, ref, secrets, dryRun, pin)
}

func localUpdate(node *p2p.QriNode, ref *repo.DatasetRef, secrets map[string]string, dryRun, pin bool) (res repo.DatasetRef, body cafs.File, err error) {
	var (
		bodyFile cafs.File
		commit   = &dataset.CommitPod{}
	)

	if ref.Dataset != nil {
		commit = ref.Dataset.Commit
	}

	if err = base.ReadDataset(node.Repo, ref); err != nil {
		log.Error(err)
		return
	}
	if ref.Dataset.Transform == nil {
		err = fmt.Errorf("transform script is required to automate updates to your own datasets")
		return
	}

	ds := &dataset.Dataset{}
	if err = ds.Decode(ref.Dataset); err != nil {
		return
	}
	ds.Commit.Title = commit.Title
	ds.Commit.Message = commit.Message

	bodyFile, err = dsfs.LoadBody(node.Repo.Store(), ds)
	if err != nil {
		log.Error(err.Error())
		return
	}

	script, err := node.Repo.Store().Get(datastore.NewKey(ds.Transform.ScriptPath))
	if err != nil {
		log.Error(err)
		return
	}
	ds.Transform.ScriptPath = script.FileName()

	node.LocalStreams.Print("🤖 executing transform\n")

	var config map[string]interface{}
	if ref.Dataset.Transform == nil {
		config = make(map[string]interface{})
	} else {
		config = ref.Dataset.Transform.Config
	}
	bodyFile, err = ExecTransform(node, ds, script, bodyFile, secrets, config, nil)
	if err != nil {
		log.Error(err)
		return
	}
	node.LocalStreams.Print("✅ transform complete\n")
	ds.PreviousPath = ref.Path

	return base.CreateDataset(node.Repo, node.LocalStreams, ref.Name, ds, bodyFile, dryRun, pin)
}

// AddDataset fetches & pins a dataset to the store, adding it to the list of stored refs
func AddDataset(node *p2p.QriNode, ref *repo.DatasetRef) (err error) {
	if !ref.Complete() {
		err = repo.CanonicalizeDatasetRef(node.Repo, ref)
		if err == nil {
			return fmt.Errorf("error: dataset %s already exists in repo", ref)
		} else if err != repo.ErrNotFound {
			return fmt.Errorf("error with new reference: %s", err.Error())
		}

		if ref.Path == "" && node != nil {
			if err := node.RequestDataset(ref); err != nil {
				return fmt.Errorf("error requesting dataset: %s", err.Error())
			}
		}
	}

	errs := make(chan error)
	tasks := 1

	rc := node.Repo.Registry()
	if rc != nil {
		tasks++
		go func() {
			ng, err := newNodeGetter(node)
			if err != nil {
				return
			}

			capi, err := newIPFSCoreAPI(node)
			if err != nil {
				return
			}

			if err := rc.DsyncFetch(node.Context(), ref.Path, ng, capi.Block()); err != nil {
				errs <- err
				return
			}
			node.LocalStreams.Print("fetched from registry")
			if pinner, ok := node.Repo.Store().(cafs.Pinner); ok {
				if err := pinner.Pin(datastore.NewKey(ref.Path), true); err != nil {
					errs <- err
				}
			}
		}()
	}

	go func() {
		errs <- base.FetchDataset(node.Repo, ref, true, true)
	}()

	success := false
	for i := 0; i < tasks; i++ {
		if err = <-errs; err == nil {
			success = true
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
	return base.SetPublishStatus(node.Repo, ref, published)
}

// RenameDataset alters a dataset name
func RenameDataset(node *p2p.QriNode, current, new *repo.DatasetRef) (err error) {
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
		return fmt.Errorf("dataset '%s/%s' already exists", new.Peername, new.Name)
	} else if err != repo.ErrNotFound {
		log.Debug(err.Error())
		return fmt.Errorf("error with new reference: %s", err.Error())
	}
	new.Path = current.Path

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
