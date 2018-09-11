package actions

import (
	"fmt"
	"strings"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// Dataset wraps a repo.Repo, adding actions for working with datasets
// type Dataset struct {
// 	Node *p2p.QriNode
// }

// CreateDataset initializes a dataset from a dataset pointer and data file
func CreateDataset(node *p2p.QriNode, name string, ds *dataset.Dataset, data cafs.File, secrets map[string]string, pin bool) (ref repo.DatasetRef, err error) {
	log.Debugf("CreateDataset: %s", name)
	var (
		r   = node.Repo
		pro *profile.Profile
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
	pro, err = r.Profile()
	if err != nil {
		return
	}

	userSet.Assign(ds)

	if ds.Commit != nil {
		// NOTE: add author ProfileID here to keep the dataset package agnostic to
		// all identity stuff except keypair crypto
		ds.Commit.Author = &dataset.User{ID: pro.ID.String()}
	}

	if ds.Transform != nil {
		log.Info("running transformation...")
		data, err = ExecTransform(node, ds, data, secrets)
		if err != nil {
			return
		}
		log.Info("done")
		ds.Assign(userSet)
	}

	if err = PrepareViz(ds); err != nil {
		return
	}

	if ref, err = repo.CreateDataset(node.Repo, name, ds, data, pin); err != nil {
		return
	}

	if err = r.LogEvent(repo.ETDsCreated, ref); err != nil {
		return
	}

	_, storeIsPinner := r.Store().(cafs.Pinner)
	if pin && storeIsPinner {
		r.LogEvent(repo.ETDsPinned, ref)
	}
	return
}

// AddDataset fetches & pins a dataset to the store, adding it to the list of stored refs
func AddDataset(node *p2p.QriNode, ref *repo.DatasetRef) (err error) {
	log.Debugf("AddDataset: %s", ref)

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

	if err = PinDataset(r, *ref); err != nil {
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

// ReadDataset grabs a dataset from the store
func ReadDataset(r repo.Repo, ref *repo.DatasetRef) (err error) {
	if store := r.Store(); store != nil {
		ds, e := dsfs.LoadDataset(store, datastore.NewKey(ref.Path))
		if e != nil {
			return e
		}
		ref.Dataset = ds.Encode()
		return
	}

	return datastore.ErrNotFound
}

// RenameDataset alters a dataset name
func RenameDataset(r repo.Repo, a, b repo.DatasetRef) (err error) {
	if err = r.DeleteRef(a); err != nil {
		return err
	}
	if err = r.PutRef(b); err != nil {
		return err
	}

	return r.LogEvent(repo.ETDsRenamed, b)
}

// PinDataset marks a dataset for retention in a store
func PinDataset(r repo.Repo, ref repo.DatasetRef) error {
	if pinner, ok := r.Store().(cafs.Pinner); ok {
		pinner.Pin(datastore.NewKey(ref.Path), true)
		return r.LogEvent(repo.ETDsPinned, ref)
	}
	return repo.ErrNotPinner
}

// UnpinDataset unmarks a dataset for retention in a store
func UnpinDataset(r repo.Repo, ref repo.DatasetRef) error {
	if pinner, ok := r.Store().(cafs.Pinner); ok {
		pinner.Unpin(datastore.NewKey(ref.Path), true)
		return r.LogEvent(repo.ETDsUnpinned, ref)
	}
	return repo.ErrNotPinner
}

// DeleteDataset removes a dataset from the store
func DeleteDataset(node *p2p.QriNode, ref repo.DatasetRef) error {
	r := node.Repo
	pro, err := r.Profile()
	if err != nil {
		return err
	}

	ds, err := dsfs.LoadDataset(r.Store(), datastore.NewKey(ref.Path))
	if err != nil {
		return err
	}

	if err = r.DeleteRef(ref); err != nil {
		return err
	}

	if rc := r.Registry(); rc != nil {
		dse := ds.Encode()
		// TODO - this should be set by LoadDataset
		dse.Path = ref.Path
		if e := rc.DeleteDataset(ref.Peername, ref.Name, dse, pro.PrivKey.GetPublic()); e != nil {
			// ignore registry errors
			log.Errorf("deleting dataset: %s", e.Error())
		}
	}

	if err = UnpinDataset(r, ref); err != nil && err != repo.ErrNotPinner {
		return err
	}

	return r.LogEvent(repo.ETDsDeleted, ref)
}
