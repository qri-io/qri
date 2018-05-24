package actions

import (
	"fmt"
	"strings"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// Dataset wraps a repo.Repo, adding actions related to working
// with datasets
type Dataset struct {
	repo.Repo
}

// CreateDataset initializes a dataset from a dataset pointer and data file
func (act Dataset) CreateDataset(name string, ds *dataset.Dataset, data cafs.File, pin bool) (ref repo.DatasetRef, err error) {
	log.Debugf("CreateDataset: %s", name)
	var (
		path datastore.Key
		pro  *profile.Profile
	)
	pro, err = act.Profile()
	if err != nil {
		return
	}

	if ds.Transform != nil && data == nil {
		log.Info("running transformation...")
		data, err = act.ExecTransform(ds)
		if err != nil {
			return
		}
		log.Info("done")
	}

	path, err = dsfs.CreateDataset(act.Store(), ds, data, act.PrivateKey(), pin)
	if err != nil {
		return
	}

	if ds.PreviousPath != "" && ds.PreviousPath != "/" {
		prev := repo.DatasetRef{
			ProfileID: pro.ID,
			Peername:  pro.Peername,
			Name:      name,
			Path:      ds.PreviousPath,
		}
		if err = act.DeleteRef(prev); err != nil {
			log.Error(err.Error())
			err = nil
		}
	}

	ref = repo.DatasetRef{
		ProfileID: pro.ID,
		Peername:  pro.Peername,
		Name:      name,
		Path:      path.String(),
	}

	if err = act.PutRef(ref); err != nil {
		log.Error(err.Error())
		return
	}

	if rc := act.Registry(); rc != nil {
		log.Debugf("posting dataset to registry: %s/%s", pro.Peername, name)
		dse := ds.Encode()
		// TODO - this should be set be dsfs.CreateDataset:
		dse.Path = path.String()

		if e := rc.PutDataset(pro.Peername, name, dse, pro.PrivKey.GetPublic()); e != nil {
			// ignore registry errors
			log.Errorf("registering dataset: %s", e.Error())
		}
	}

	if err = act.LogEvent(repo.ETDsCreated, ref); err != nil {
		return
	}

	_, storeIsPinner := act.Store().(cafs.Pinner)
	if pin && storeIsPinner {
		act.LogEvent(repo.ETDsPinned, ref)
	}
	return
}

// AddDataset fetches & pins a dataset to the store, adding it to the list of stored refs
// TODO - this needs tests, first we need an implementation of the fetcher interface that isn't cafs/ipfs
func (act Dataset) AddDataset(ref *repo.DatasetRef) (err error) {
	log.Debugf("AddDataset: %s", ref)
	fetcher, ok := act.Store().(cafs.Fetcher)
	if !ok {
		err = fmt.Errorf("this store cannot fetch from remote sources")
		return
	}

	key := datastore.NewKey(strings.TrimSuffix(ref.Path, "/"+dsfs.PackageFileDataset.String()))
	_, err = fetcher.Fetch(cafs.SourceAny, key)
	if err != nil {
		return fmt.Errorf("error fetching file: %s", err.Error())
	}

	if err = act.PinDataset(*ref); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error pinning root key: %s", err.Error())
	}

	if err = act.PutRef(*ref); err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error putting dataset name in repo: %s", err.Error())
	}

	path := datastore.NewKey(key.String() + "/" + dsfs.PackageFileDataset.String())
	ds, err := dsfs.LoadDataset(act.Store(), path)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error loading newly saved dataset path: %s", path.String())
	}

	ref.Dataset = ds.Encode()
	return
}

// ReadDataset grabs a dataset from the store
func (act Dataset) ReadDataset(ref *repo.DatasetRef) (err error) {
	if act.Repo.Store() != nil {
		ds, e := dsfs.LoadDataset(act.Store(), datastore.NewKey(ref.Path))
		if e != nil {
			return e
		}
		ref.Dataset = ds.Encode()
		return
	}

	return datastore.ErrNotFound
}

// RenameDataset alters a dataset name
func (act Dataset) RenameDataset(a, b repo.DatasetRef) (err error) {
	if err = act.DeleteRef(a); err != nil {
		return err
	}
	if err = act.PutRef(b); err != nil {
		return err
	}

	return act.LogEvent(repo.ETDsRenamed, b)
}

// PinDataset marks a dataset for retention in a store
func (act Dataset) PinDataset(ref repo.DatasetRef) error {
	if pinner, ok := act.Store().(cafs.Pinner); ok {
		pinner.Pin(datastore.NewKey(ref.Path), true)
		return act.LogEvent(repo.ETDsPinned, ref)
	}
	return repo.ErrNotPinner
}

// UnpinDataset unmarks a dataset for retention in a store
func (act Dataset) UnpinDataset(ref repo.DatasetRef) error {
	if pinner, ok := act.Store().(cafs.Pinner); ok {
		pinner.Unpin(datastore.NewKey(ref.Path), true)
		return act.LogEvent(repo.ETDsUnpinned, ref)
	}
	return repo.ErrNotPinner
}

// DeleteDataset removes a dataset from the store
func (act Dataset) DeleteDataset(ref repo.DatasetRef) error {
	pro, err := act.Profile()
	if err != nil {
		return err
	}

	ds, err := dsfs.LoadDataset(act.Store(), datastore.NewKey(ref.Path))
	if err != nil {
		return err
	}

	if err = act.DeleteRef(ref); err != nil {
		return err
	}

	if rc := act.Registry(); rc != nil {
		dse := ds.Encode()
		// TODO - this should be set by LoadDataset
		dse.Path = ref.Path
		if e := rc.DeleteDataset(ref.Peername, ref.Name, dse, pro.PrivKey.GetPublic()); e != nil {
			// ignore registry errors
			log.Errorf("deleting dataset: %s", e.Error())
		}
	}

	if err = act.UnpinDataset(ref); err != nil && err != repo.ErrNotPinner {
		return err
	}

	return act.LogEvent(repo.ETDsDeleted, ref)
}
