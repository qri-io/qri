package base

import (
	"context"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

type TestRunner struct {
	Repo    repo.Repo
	Context context.Context
}

func newTestRunner(t *testing.T) *TestRunner {
	ctx := context.Background()
	r := newTestRepo(t)
	return &TestRunner{Context: ctx, Repo: r}
}

func (run *TestRunner) Delete() {
}

func (run *TestRunner) BuildDataset(dsName, bodyFormat string) *dataset.Dataset {
	ds := dataset.Dataset{
		Peername: "peer",
		Name:     dsName,
		Structure: &dataset.Structure{
			Format: bodyFormat,
			Schema: map[string]interface{}{"type": "array"},
		},
	}
	return &ds
}

func (run *TestRunner) SaveDataset(ds *dataset.Dataset) (dsref.Ref, error) {
	sw := SaveSwitches{}
	return run.saveDataset(ds, sw)
}

func (run *TestRunner) SaveDatasetDryRun(ds *dataset.Dataset) (dsref.Ref, error) {
	sw := SaveSwitches{DryRun: true}
	return run.saveDataset(ds, sw)
}

func (run *TestRunner) SaveDatasetReplace(ds *dataset.Dataset) (dsref.Ref, error) {
	sw := SaveSwitches{Replace: true}
	return run.saveDataset(ds, sw)
}

func (run *TestRunner) saveDataset(ds *dataset.Dataset, sw SaveSwitches) (dsref.Ref, error) {
	headRef := ""
	book := run.Repo.Logbook()
	initID, err := book.RefToInitID(dsref.Ref{Username: "peer", Name: ds.Name})
	if err == nil {
		got, _ := run.Repo.GetRef(reporef.DatasetRef{Peername: "peer", Name: ds.Name})
		headRef = got.Path
	} else if err == logbook.ErrNotFound {
		initID, err = book.WriteDatasetInit(run.Context, ds.Name)
	}
	if err != nil {
		return dsref.Ref{}, err
	}
	datasetRef, err := SaveDataset(run.Context, run.Repo, devNull, initID, headRef, ds, nil, nil, sw)
	return reporef.ConvertToDsref(datasetRef), err
}
