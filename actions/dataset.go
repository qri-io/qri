package actions

import (
	"fmt"
	"os"

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
func SaveDataset(node *p2p.QriNode, dsp *dataset.DatasetPod, dryRun, pin bool) (ref repo.DatasetRef, body cafs.File, err error) {
	var (
		ds                       *dataset.Dataset
		prevPath                 string
		bodyFile, changeBodyFile cafs.File
		secrets                  map[string]string
		pro                      *profile.Profile
		changes                  = &dataset.Dataset{}
		r                        = node.Repo
	)

	// set ds to dataset head, or empty dataset if no history
	ds, bodyFile, prevPath, err = base.PrepareDatasetSave(r, dsp.Peername, dsp.Name)
	if err != nil {
		return
	}

	if dryRun {
		node.LocalStreams.Print("🏃🏽‍♀️ dry run\n")

		pro, err = r.Profile()
		if err != nil {
			return
		}

		// dry-runs store to an in-memory repo
		r, err = repo.NewMemRepo(pro, cafs.NewMapstore(), profile.NewMemStore(), nil)
		if err != nil {
			return
		}
	}

	if changeBodyFile, err = base.DatasetPodBodyFile(dsp); err == nil && changeBodyFile != nil {
		dsp.BodyPath = ""
		bodyFile = changeBodyFile
	} else if err != nil {
		return
	}

	if err = changes.Decode(dsp); err != nil {
		return
	}
	ds.Assign(changes)
	clearPaths(ds)

	if ds.Transform != nil {
		mutateCheck := mutatedComponentsFunc(dsp)
		if ds.Transform.Script == nil {
			var f *os.File
			f, err = os.Open(ds.Transform.ScriptPath)
			if err != nil {
				return
			}
			ds.Transform.Script = f
		}
		// TODO - consider making this a standard method on dataset.Transform
		script := cafs.NewMemfileReader(ds.Transform.ScriptPath, ds.Transform.Script)

		node.LocalStreams.Print("🤖 executing transform\n")
		bodyFile, err = ExecTransform(node, ds, script, bodyFile, secrets, mutateCheck)
		if err != nil {
			return
		}
		node.LocalStreams.Print("✅ transform complete\n")
	}

	// let's make history, if it exists:
	ds.PreviousPath = prevPath
	return base.CreateDataset(r, node.LocalStreams, dsp.Name, ds, bodyFile, dryRun, pin)
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
func UpdateDataset(node *p2p.QriNode, ref *repo.DatasetRef, dryRun, pin bool) (res repo.DatasetRef, body cafs.File, err error) {
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

	return localUpdate(node, ref, dryRun, pin)
}

func localUpdate(node *p2p.QriNode, ref *repo.DatasetRef, dryRun, pin bool) (res repo.DatasetRef, body cafs.File, err error) {
	var (
		bodyFile cafs.File
		secrets  map[string]string
		commit   = &dataset.CommitPod{}
	)

	if ref.Dataset != nil {
		commit = ref.Dataset.Commit
		if ref.Dataset.Transform != nil {
			secrets = ref.Dataset.Transform.Secrets
		}
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
	bodyFile, err = ExecTransform(node, ds, script, bodyFile, secrets, nil)
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

	if err = base.FetchDataset(node.Repo, ref, true, true); err != nil {
		return
	}

	if err = node.Repo.PutRef(*ref); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error putting dataset name in repo: %s", err.Error())
	}

	return nil
}

// SetPublishStatus configures the publish status of a stored reference
func SetPublishStatus(node *p2p.QriNode, ref *repo.DatasetRef, published bool) (err error) {
	// currently we're just passing this call off to the base package,
	// in the near future we'll start publishing to registries here
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
