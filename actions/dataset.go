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
)

// SaveDataset initializes a dataset from a dataset pointer and data file
func SaveDataset(node *p2p.QriNode, dsp *dataset.DatasetPod, dryRun, pin bool) (ref repo.DatasetRef, body cafs.File, err error) {
	var (
		ds       *dataset.Dataset
		bodyFile cafs.File
		secrets  map[string]string
		// NOTE - struct fields need to be instantiated to make assign set to
		// new pointer values
		userSet = &dataset.Dataset{
			Commit:    &dataset.Commit{},
			Meta:      &dataset.Meta{},
			Structure: &dataset.Structure{},
			Transform: &dataset.Transform{},
			Viz:       &dataset.Viz{},
		}
	)

	if dryRun {
		node.LocalStreams.Print("🏃🏽‍♀️ dry run\n")
	}

	// Determine if the save is creating a new dataset or updating an existing dataset by
	// seeing if the name can canonicalize to a repo that we know about
	lookup := &repo.DatasetRef{Name: dsp.Name, Peername: dsp.Peername}
	err = repo.CanonicalizeDatasetRef(node.Repo, lookup)
	if err == repo.ErrNotFound {
		ds, bodyFile, secrets, err = base.PrepareDatasetNew(dsp)
		if err != nil {
			return
		}
	} else {
		ds, bodyFile, secrets, err = base.PrepareDatasetSave(node.Repo, dsp)
		if err != nil {
			return
		}
	}

	userSet.Assign(ds)

	if ds.Transform != nil {
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
		bodyFile, err = ExecTransform(node, ds, script, bodyFile, secrets)
		if err != nil {
			return
		}
		node.LocalStreams.Print("✅ transform complete\n")
		ds.Assign(userSet)
	}

	return base.CreateDataset(node.Repo, node.LocalStreams, dsp.Name, ds, bodyFile, dryRun, pin)
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
		err = fmt.Errorf("remote updates are not yet finished")
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
	bodyFile, err = ExecTransform(node, ds, script, bodyFile, secrets)
	if err != nil {
		log.Error(err)
		return
	}
	node.LocalStreams.Print("✅ transform complete\n")

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

	r := node.Repo
	key := datastore.NewKey(strings.TrimSuffix(ref.Path, "/"+dsfs.PackageFileDataset.String()))
	path := datastore.NewKey(key.String() + "/" + dsfs.PackageFileDataset.String())

	fetcher, ok := r.Store().(cafs.Fetcher)
	if !ok {
		err = fmt.Errorf("this store cannot fetch from remote sources")
		return
	}

	// TODO: This is asserting that the target is Fetch-able, but inside dsfs.LoadDataset,
	// only Get is called. Clean up the semantics of Fetch and Get to get this expection
	// more correctly in line with what's actually required.
	_, err = fetcher.Fetch(cafs.SourceAny, key)
	if err != nil {
		return fmt.Errorf("error fetching file: %s", err.Error())
	}

	if err = base.PinDataset(r, *ref); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error pinning root key: %s", err.Error())
	}

	if err = r.PutRef(*ref); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error putting dataset name in repo: %s", err.Error())
	}

	ds, err := dsfs.LoadDataset(r.Store(), path)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error loading newly saved dataset path: %s", path.String())
	}

	ref.Dataset = ds.Encode()
	return
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

	// ds, err := dsfs.LoadDataset(r.Store(), datastore.NewKey(ref.Path))
	// if err != nil {
	// 	return err
	// }

	if err = r.DeleteRef(*ref); err != nil {
		return err
	}

	if err = base.UnpinDataset(r, *ref); err != nil && err != repo.ErrNotPinner {
		return err
	}

	return r.LogEvent(repo.ETDsDeleted, *ref)
}
