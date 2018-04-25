package actions

import (
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
	var (
		path datastore.Key
		pro  *profile.Profile
	)
	pro, err = act.Profile()
	if err != nil {
		return
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

	if err = act.LogEvent(repo.ETDsCreated, ref); err != nil {
		return
	}

	_, storeIsPinner := act.Store().(cafs.Pinner)
	if pin && storeIsPinner {
		act.LogEvent(repo.ETDsPinned, ref)
	}
	return
}

// ReadDataset grabs a dataset from the store
func (act Dataset) ReadDataset(ref *repo.DatasetRef) (err error) {
	if act.Repo.Store() != nil {
		ds, e := dsfs.LoadDataset(act.Store(), datastore.NewKey(ref.Path))
		if err != nil {
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
	if err := act.DeleteRef(ref); err != nil {
		return err
	}
	if err := act.UnpinDataset(ref); err != nil && err != repo.ErrNotPinner {
		return err
	}

	return act.LogEvent(repo.ETDsDeleted, ref)
}
