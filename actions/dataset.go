package actions

import (
	"fmt"
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

// TODO: Move this to core.CreateDataset

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
		node.LocalStreams.Print("🤖 executing transform\n")
		bodyFile, err = ExecTransform(node, ds, bodyFile, secrets)
		if err != nil {
			return
		}
		node.LocalStreams.Print("✅ transform complete\n")
		ds.Assign(userSet)
	}

	return base.CreateDataset(node.Repo, node.LocalStreams, dsp.Name, ds, bodyFile, dryRun, pin)
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
