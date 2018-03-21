package repo

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
)

// // MemDatasets is an in-memory implementation of the DatasetStore interface
// type MemDatasets struct {
// 	datasets map[string]*dataset.Dataset
// 	store    cafs.Filestore
// }

// // NewMemDatasets creates a datasets instance from a cafs.Filstore
// func NewMemDatasets(store cafs.Filestore) MemDatasets {
// 	return MemDatasets{
// 		datasets: map[string]*dataset.Dataset{},
// 		store:    store,
// 	}
// }

// CreateDataset initializes a dataset from a dataset pointer and data file
func (r *MemRepo) CreateDataset(name string, ds *dataset.Dataset, data cafs.File, pin bool) (ref DatasetRef, err error) {
	var (
		path datastore.Key
		pro  = r.profile
	)

	path, err = dsfs.CreateDataset(r.store, ds, data, r.pk, pin)
	if err != nil {
		return
	}

	if ds.PreviousPath != "" && ds.PreviousPath != "/" {
		prev := DatasetRef{
			PeerID:   pro.ID,
			Peername: pro.Peername,
			Name:     name,
			Path:     ds.PreviousPath,
		}
		if err = r.DeleteRef(prev); err != nil {
			// log.Error(err.Error())
			err = nil
		}
	}

	ref = DatasetRef{
		Peername: r.profile.Peername,
		Name:     name,
		PeerID:   r.profile.ID,
		Path:     path.String(),
	}

	if err = r.PutRef(ref); err != nil {
		return ref, err
	}

	if err = r.LogEvent(ETDsCreated, ref); err != nil {
		return ref, err
	}

	if pin {
		if err = r.LogEvent(ETDsPinned, ref); err != nil {
			return ref, err
		}
	}
	return ref, nil
}

// ReadDataset fetches a dataset from the store
func (r *MemRepo) ReadDataset(ref *DatasetRef) error {
	ds, err := dsfs.LoadDataset(r.store, datastore.NewKey(ref.Path))
	if err != nil {
		return err
	}
	ref.Dataset = ds
	return nil
}

// RenameDataset alters a dataset name
func (r *MemRepo) RenameDataset(a, b DatasetRef) (err error) {
	if err = r.DeleteRef(a); err != nil {
		return err
	}
	if err = r.PutRef(b); err != nil {
		return err
	}
	return r.LogEvent(ETDsRenamed, b)
}

// PinDataset marks a dataset for retention in a store
func (r *MemRepo) PinDataset(ref DatasetRef) (err error) {
	if pinner, ok := r.store.(cafs.Pinner); ok {
		pinner.Pin(datastore.NewKey(ref.Path), true)
		return r.LogEvent(ETDsUnpinned, ref)
	}
	return ErrNotPinner
}

// UnpinDataset unmarks a dataset for retention in a store
func (r *MemRepo) UnpinDataset(ref DatasetRef) error {
	if pinner, ok := r.store.(cafs.Pinner); ok {
		pinner.Unpin(datastore.NewKey(ref.Path), true)
		return r.LogEvent(ETDsUnpinned, ref)
	}
	return ErrNotPinner
}

// DeleteDataset removes a dataset from the store
func (r *MemRepo) DeleteDataset(ref DatasetRef) error {
	if err := r.DeleteRef(ref); err != nil {
		return err
	}
	if err := r.UnpinDataset(ref); err != nil && err != ErrNotPinner {
		return err
	}

	return r.LogEvent(ETDsDeleted, ref)
}
