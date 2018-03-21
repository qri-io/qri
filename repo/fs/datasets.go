package fsrepo

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// CreateDataset initializes a dataset from a dataset pointer and data file
func (r *Repo) CreateDataset(name string, ds *dataset.Dataset, data cafs.File, pin bool) (ref repo.DatasetRef, err error) {
	var (
		path datastore.Key
		pro  *profile.Profile
	)
	pro, err = r.Profile()
	if err != nil {
		return
	}

	path, err = dsfs.CreateDataset(r.store, ds, data, r.pk, pin)
	if err != nil {
		return
	}

	if ds.PreviousPath != "" && ds.PreviousPath != "/" {
		prev := repo.DatasetRef{
			PeerID:   pro.ID,
			Peername: pro.Peername,
			Name:     name,
			Path:     ds.PreviousPath,
		}
		if err = r.Refstore.DeleteRef(prev); err != nil {
			log.Error(err.Error())
			err = nil
		}
	}

	ref = repo.DatasetRef{
		PeerID:   pro.ID,
		Peername: pro.Peername,
		Name:     name,
		Path:     path.String(),
	}

	if err = r.PutRef(ref); err != nil {
		log.Error(err.Error())
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

// ReadDataset grabs a dataset from the store
func (r *Repo) ReadDataset(ref *repo.DatasetRef) (err error) {
	if r.store != nil {
		ref.Dataset, err = dsfs.LoadDataset(r.store, datastore.NewKey(ref.Path))
		return
	}

	return datastore.ErrNotFound
}

// RenameDataset alters a dataset name
func (r *Repo) RenameDataset(a, b repo.DatasetRef) (err error) {
	if err = r.Refstore.DeleteRef(a); err != nil {
		return err
	}
	if err = r.Refstore.PutRef(b); err != nil {
		return err
	}

	return r.LogEvent(repo.ETDsRenamed, b)
}

// PinDataset marks a dataset for retention in a store
func (r *Repo) PinDataset(ref repo.DatasetRef) error {
	if pinner, ok := r.store.(cafs.Pinner); ok {
		pinner.Pin(datastore.NewKey(ref.Path), true)
		return r.LogEvent(repo.ETDsPinned, ref)
	}
	return repo.ErrNotPinner
}

// UnpinDataset unmarks a dataset for retention in a store
func (r *Repo) UnpinDataset(ref repo.DatasetRef) error {
	if pinner, ok := r.store.(cafs.Pinner); ok {
		pinner.Unpin(datastore.NewKey(ref.Path), true)
		return r.LogEvent(repo.ETDsUnpinned, ref)
	}
	return repo.ErrNotPinner
}

// DeleteDataset removes a dataset from the store
func (r *Repo) DeleteDataset(ref repo.DatasetRef) error {
	if err := r.Refstore.DeleteRef(ref); err != nil {
		return err
	}
	if err := r.UnpinDataset(ref); err != nil && err != repo.ErrNotPinner {
		return err
	}

	return r.LogEvent(repo.ETDsDeleted, ref)
}
