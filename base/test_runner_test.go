package base

import (
	"context"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
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
	sw := SaveSwitches{}
	return run.saveDataset(ds, sw)
}

func (run *TestRunner) SaveDatasetReplace(ds *dataset.Dataset) (dsref.Ref, error) {
	sw := SaveSwitches{Replace: true}
	return run.saveDataset(ds, sw)
}

func (run *TestRunner) saveDataset(ds *dataset.Dataset, sw SaveSwitches) (dsref.Ref, error) {
	book := run.Repo.Logbook()
	author := book.Owner()
	ref := dsref.Ref{Username: author.Peername, Name: ds.Name}
	if _, err := book.ResolveRef(context.Background(), &ref); err == dsref.ErrRefNotFound {
		ref.InitID, err = book.WriteDatasetInit(run.Context, author, ds.Name)
		if err != nil {
			return dsref.Ref{}, err
		}
	} else if err != nil {
		return dsref.Ref{}, err
	}

	ds, err := SaveDataset(run.Context, run.Repo, run.Repo.Filesystem().DefaultWriteFS(), run.Repo.Profiles().Owner(), ref.InitID, ref.Path, ds, nil, sw)
	if err != nil {
		return dsref.Ref{}, err
	}
	return dsref.ConvertDatasetToVersionInfo(ds).SimpleRef(), nil
}
