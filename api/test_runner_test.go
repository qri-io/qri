package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/automation"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/automation/workflow/wftest"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/profile"
)

type APITestRunner struct {
	cancelCtx     context.CancelFunc
	Ctx           context.Context
	Node          *p2p.QriNode
	NodeTeardown  func()
	Inst          *lib.Instance
	workflowStore workflow.Store
	DsfsTsFunc    func() time.Time
	TmpDir        string
	WorkDir       string
	PrevXformVer  string
}

func NewAPITestRunner(t *testing.T) *APITestRunner {
	ctx, cancel := context.WithCancel(context.Background())
	run := APITestRunner{
		cancelCtx: cancel,
		Ctx:       ctx,
	}
	run.Node, run.NodeTeardown = newTestNode(t)
	o := automation.DefaultMemOrchestratorOptions(ctx, event.NilBus)
	run.workflowStore = o.WorkflowStore

	run.Inst = newTestInstanceWithProfileFromNodeAndOrchestratorOpts(ctx, run.Node, &o)

	wftest.MustAddDefaultWorkflows(t, &run)

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

func (r *APITestRunner) Instance() *lib.Instance {
	return r.Inst
}

func (r *APITestRunner) Owner() *profile.Profile {
	return r.Node.Repo.Profiles().Owner()
}

func (r *APITestRunner) Context() context.Context {
	return r.Ctx
}

func (r *APITestRunner) WorkflowStore() workflow.Store {
	return r.workflowStore
}

func (r *APITestRunner) Delete() {
	os.RemoveAll(r.TmpDir)
	os.RemoveAll("/tmp/qri_api_test")
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
	saveParams := lib.SaveParams{
		Ref:      fmt.Sprintf("peer/%s", ds.Name),
		Dataset:  ds,
		BodyPath: bodyFilename,
	}
	_, err := r.Inst.Dataset().Save(r.Ctx, &saveParams)
	if err != nil {
		panic(err)
	}
}

func (r *APITestRunner) MustTestServer(t *testing.T) *httptest.Server {
	s := New(r.Inst)
	return httptest.NewServer(NewServerRoutes(s))
}
