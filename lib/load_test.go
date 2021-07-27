package lib

import (
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/event"
)

func TestLoadDataset(t *testing.T) {
	tr := newTestRunner(t)
	defer tr.Delete()

	fs := tr.Instance.Repo().Filesystem()

	if _, err := (*datasetLoader)(nil).LoadDataset(tr.Ctx, ""); err == nil {
		t.Errorf("expected loadDataset on a nil instance to fail without panicing")
	}

	loader := &datasetLoader{inst: nil}
	if _, err := loader.LoadDataset(tr.Ctx, ""); err == nil {
		t.Errorf("expected loadDataset on a nil instance to fail without panicing")
	}

	loader = &datasetLoader{inst: tr.Instance}
	dsrefspec.AssertLoaderSpec(t, loader, func(ds *dataset.Dataset) (*dsref.Ref, error) {
		// Allocate an initID for this dataset
		username := tr.Instance.repo.Profiles().Owner().Peername
		initID, err := tr.Instance.logbook.WriteDatasetInit(tr.Ctx, username, ds.Name)
		if err != nil {
			return nil, err
		}
		// Create the dataset in the provided storage
		ref := &dsref.Ref{
			InitID:   initID,
			Username: username,
			Name:     ds.Name,
		}
		path, err := dsfs.CreateDataset(
			tr.Ctx,
			fs,
			fs.DefaultWriteFS(),
			event.NilBus,
			ds,
			nil,
			tr.Instance.repo.Profiles().Owner().PrivKey,
			dsfs.SaveSwitches{},
		)
		if err != nil {
			return nil, err
		}
		// Save the reference that the loader will use to laod
		ref.Path = path
		ds.Path = path
		if err = tr.Instance.logbook.WriteVersionSave(tr.Ctx, initID, ds, nil); err != nil {
			return nil, err
		}
		return ref, nil
	})
}
