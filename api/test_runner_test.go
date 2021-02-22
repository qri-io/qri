package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
)

type APITestRunner struct {
	cancelCtx    context.CancelFunc
	Ctx          context.Context
	Node         *p2p.QriNode
	NodeTeardown func()
	Inst         *lib.Instance
	DsfsTsFunc   func() time.Time
	TmpDir       string
	WorkDir      string
	PrevXformVer string
}

func NewAPITestRunner(t *testing.T) *APITestRunner {
	ctx, cancel := context.WithCancel(context.Background())
	run := APITestRunner{
		cancelCtx: cancel,
		Ctx:       ctx,
	}
	run.Node, run.NodeTeardown = newTestNode(t)
	run.Inst = newTestInstanceWithProfileFromNode(ctx, run.Node)

	tmpDir, err := ioutil.TempDir("", "api_test")
	if err != nil {
		t.Fatal(err)
	}
	run.TmpDir = tmpDir

	counter := 0
	run.DsfsTsFunc = dsfs.Timestamp
	dsfs.Timestamp = func() time.Time {
		counter++
		return time.Date(2001, 01, 01, 01, counter, 01, 01, time.UTC)
	}

	run.PrevXformVer = APIVersion
	APIVersion = "test_version"

	return &run
}

func (r *APITestRunner) Delete() {
	os.RemoveAll(r.TmpDir)
	APIVersion = r.PrevXformVer
	r.cancelCtx()
	r.NodeTeardown()
}

func (r *APITestRunner) MustMakeWorkDir(t *testing.T, name string) string {
	r.WorkDir = filepath.Join(r.TmpDir, name)
	if err := os.MkdirAll(r.WorkDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	return r.WorkDir
}

func (r *APITestRunner) BuildDataset(dsName string) *dataset.Dataset {
	ds := dataset.Dataset{
		Peername: "peer",
		Name:     dsName,
	}
	return &ds
}

func (r *APITestRunner) SaveDataset(ds *dataset.Dataset, bodyFilename string) {
	dsm := lib.NewDatasetMethods(r.Inst)
	saveParams := lib.SaveParams{
		Ref:      fmt.Sprintf("peer/%s", ds.Name),
		Dataset:  ds,
		BodyPath: bodyFilename,
	}
	_, err := dsm.Save(r.Ctx, &saveParams)
	if err != nil {
		panic(err)
	}
}

func (r *APITestRunner) NewRenderHandlers() *RenderHandlers {
	return NewRenderHandlers(r.Inst)
}
