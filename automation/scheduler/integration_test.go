package scheduler

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/iso8601"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/lib"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestScheduleWorkflowIntegration(t *testing.T) {

	username := "integration_test"
	tr, cleanup := NewSchedulerTestRunner(t, username)
	defer cleanup()
	workflowName := "workflowName"
	ownerID := "ownerID"
	// TODO (ramfox): until we replace `datasetID` with `InitID`, qrimatic
	// expects the datasetID to be the dataset alias.
	// TODO (ramfox): until we get multi tenancy the only "username" that
	// qri will expect is the username of the repo
	datasetID := fmt.Sprintf("%s/dataset_name", username)
	w, err := workflow.NewCronWorkflow(workflowName, ownerID, datasetID, "R/PT1S")
	if err != nil {
		t.Fatalf("creating workflow: %s", err)
	}
	dp := &DeployParams{
		Apply:    true,
		Workflow: w,
		Transform: &dataset.Transform{
			Steps: []*dataset.TransformStep{
				{
					Name:   "transform",
					Syntax: "starlark",
					Script: `def transform(ds,ctx):
	b = ds.get_body()
	if not b:
		ds.set_body([[1,2,3]])
	else:
		b = b + [[b[len(b)-1][0] + 3, b[len(b)-1][1] + 3, b[len(b)-1][2] +3]]
		ds.set_body(b)`,
					Category: "transform",
				},
			},
		},
	}
	ctx := context.Background()
	// Deploy saves a version of the dataset & schedules the workflow
	if _, err := tr.cron.Deploy(ctx, tr.inst, dp); err != nil {
		t.Fatalf("deploying workflow: %s", err)
	}

	// ensure workflow is in the cron store
	deployedWorkflow, err := tr.cron.Workflow(ctx, w.ID)
	if err != nil {
		t.Fatalf("getting workflow from cron: %s", err)
	}
	if w.ID != deployedWorkflow.ID {
		t.Errorf("deployed workflow ID mismatch: expected %q, got %q", w.ID, deployedWorkflow.ID)
	}
	if deployedWorkflow.RunCount != 1 {
		t.Errorf("deployed workflow run count mismatch: expected 1, got %d", deployedWorkflow.RunCount)
	}

	// fetch the dataset & compare bodies
	AssertBodyEquals(t, tr.inst, datasetID, "[[1,2,3]]")

	// manually trigger workflow
	tr.cron.RunWorkflow(ctx, w, w.Triggers[0].Info().ID)
	manuallyTriggered, err := tr.cron.Workflow(ctx, w.ID)
	if manuallyTriggered.RunCount != 2 {
		t.Errorf("manually triggered workflow run count mismatch: expected 2, got %d", manuallyTriggered.RunCount)
	}

	// fetch the dataset and compare bodies
	AssertBodyEquals(t, tr.inst, datasetID, "[[1,2,3],[4,5,6]]")

	// update workflow to have new trigger time of 3 seconds
	ri, err := iso8601.ParseRepeatingInterval("R/PT3S")
	if err != nil {
		t.Fatal(err)
	}
	newTrigger := workflow.NewCronTrigger(w.ID, time.Now(), ri)
	w.Triggers = []workflow.Trigger{newTrigger}
	if err := tr.cron.UpdateWorkflow(ctx, w); err != nil {
		t.Fatal(err)
	}
	// check that trigger time has updated in store
	updatedWF, err := tr.cron.Workflow(ctx, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(newTrigger, updatedWF.Triggers[0], cmp.Comparer(workflow.CompareDurations)); diff != "" {
		t.Errorf("triggers mismatch (-want +got):\n%s", diff)
	}
	// start cron service
	cancelCtx, cancel := context.WithCancel(ctx)

	go func() {
		// give trigger enough time to run
		<-time.After(5 * time.Second)
		cancel()
	}()

	if err := tr.cron.Start(cancelCtx); err != nil {
		t.Fatal(err)
	}

	// fetch workflow, see that it has triggered
	cronTriggered, err := tr.cron.Workflow(ctx, w.ID)
	if err != nil {
		t.Fatalf("getting workflow from cron: %s", err)
	}
	if cronTriggered.RunCount != 3 {
		t.Errorf("cron triggered workflow run count mismatch: expected 3, got %d", cronTriggered.RunCount)
	}

	// fetch the dataset and compare bodies
	AssertBodyEquals(t, tr.inst, datasetID, "[[1,2,3],[4,5,6],[7,8,9]]")

	// update transform in workflow & deploy
	dp = &DeployParams{
		Apply:    true,
		Workflow: cronTriggered,
		Transform: &dataset.Transform{
			Steps: []*dataset.TransformStep{
				{
					Name:   "transform",
					Syntax: "starlark",
					Script: `def transform(ds,ctx):
	ds.set_body([["one","two","three"]])`,
					Category: "transform",
				},
			},
		},
	}
	// Deploy saves a version of the dataset & schedules the workflow
	_, err = tr.cron.Deploy(ctx, tr.inst, dp)
	if err != nil {
		t.Fatalf("deploying workflow: %s", err)
	}
	// ensure workflow is in the cron store
	transformUpdatedWF, err := tr.cron.Workflow(ctx, w.ID)
	if err != nil {
		t.Fatalf("getting workflow from cron: %s", err)
	}
	if transformUpdatedWF.RunCount != 4 {
		t.Errorf("transform updated workflow run count mismatch: expected 4, got %d", transformUpdatedWF.RunCount)
	}

	// check dataset body is different
	AssertBodyEquals(t, tr.inst, datasetID, "[[\"one\",\"two\",\"three\"]]")

	// undeploy & show there is no workflow
	if err := tr.cron.Undeploy(ctx, w.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.cron.Workflow(ctx, w.ID); err != workflow.ErrNotFound {
		t.Errorf("error: getting a Workflow that has been undeployed should come back with 'ErrNotFound', got nil error instead")
	}
}

// copy/pasted from `api` (can't be imported b/c of circular import issues):
// newInstanceRunnerFactory returns a factory function that produces a workflow
// runner from a qri instance
func newInstanceRunnerFactory(inst *lib.Instance) func(ctx context.Context) RunWorkflowFunc {
	return func(ctx context.Context) RunWorkflowFunc {
		return func(ctx context.Context, streams ioes.IOStreams, workflow *workflow.Workflow) error {
			runID := run.NewID()
			p := &lib.SaveParams{
				Ref: workflow.DatasetID,
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						RunID: runID,
					},
				},
				Apply: true,
			}
			_, err := inst.Dataset().Save(ctx, p)
			return err
		}
	}
}

type SchedulerTestRunner struct {
	cancel     context.CancelFunc
	TestCrypto key.CryptoGenerator
	repo       *repotest.TempRepo
	inst       *lib.Instance
	store      *workflow.Store
	cron       *Cron
	storePath  string
	// when we get identity that is separate from the qri repo identity
	// we need to keep track of the different users creating workflows in the
	// test runner
	// maybe with some `User` struct that keeps a username and a generated
	// private key, that gets connected correctly to the qrimatic and qri instances
	// beth &User
}

func (tr *SchedulerTestRunner) cleanup() {
	tr.cancel()
	if tr.storePath != "" {
		os.RemoveAll(tr.storePath)
	}
	if tr.repo != nil {
		tr.repo.Delete()
	}
}

func NewSchedulerTestRunner(t *testing.T, prefix string) (*SchedulerTestRunner, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	tr := &SchedulerTestRunner{
		cancel:     cancel,
		TestCrypto: repotest.NewTestCrypto(),
	}
	repo, err := repotest.NewTempRepo(prefix, fmt.Sprintf("%s_scheduler_test_runner_repo", prefix), tr.TestCrypto)
	if err != nil {
		t.Fatal(err)
	}
	tr.repo = &repo

	tr.inst, err = lib.NewInstance(ctx, tr.repo.QriPath, lib.OptIOStreams(ioes.NewDiscardIOStreams()))
	if err != nil {
		t.Fatal(err)
	}

	tr.storePath, err = ioutil.TempDir("", fmt.Sprintf("%s_scheduler_test_runner_store", prefix))
	if err != nil {
		t.Fatal(err)
	}

	store, err := workflow.NewFileStore(filepath.Join(tr.storePath, "workflows.json"), tr.inst.Bus())
	if err != nil {
		t.Fatal(err)
	}
	tr.store = &store

	f := newInstanceRunnerFactory(tr.inst)
	tr.cron = NewCronScheduler(*tr.store, f, tr.inst.Bus())
	return tr, tr.cleanup
}

func AssertBodyEquals(t *testing.T, inst *lib.Instance, refstr, expectBody string) {
	t.Helper()
	getParams := &lib.GetParams{
		Refstr:   refstr,
		Selector: "body",
		All:      true,
	}
	getRes, err := inst.Dataset().Get(context.Background(), getParams)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectBody, string(getRes.Bytes)); diff != "" {
		t.Errorf("dataset body mismatch (-want +got):\n%s", diff)
	}
}
