package scheduler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/iso8601"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/lib"
	repotest "github.com/qri-io/qri/repo/test"
)

func mustRepeatingInterval(s string) iso8601.RepeatingInterval {
	ri, err := iso8601.ParseRepeatingInterval(s)
	if err != nil {
		panic(err)
	}
	return ri
}

func TestCronDataset(t *testing.T) {
	updateCount := 0
	wf := &workflow.Workflow{
		Name:      "b5/libp2p_node_count",
		DatasetID: "dsID",
		OwnerID:   "ownerID",
		Type:      workflow.JTDataset,
	}

	factory := func(outer context.Context) RunWorkflowFunc {
		return func(ctx context.Context, streams ioes.IOStreams, wf *workflow.Workflow) error {
			switch wf.Type {
			case workflow.JTDataset:
				updateCount++
				return nil
			}
			t.Fatalf("runner called with invalid workflow: %v", wf)
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*1000)
	defer cancel()

	store := workflow.NewMemStore(event.NilBus)
	cron := NewCronInterval(store, factory, event.NilBus, time.Millisecond*50)
	if err := cron.Schedule(ctx, wf); err != nil {
		t.Fatal(err)
	}

	if err := cron.Start(ctx); err != nil {
		t.Fatal(err)
	}

	<-ctx.Done()

	expectedUpdateCount := 1
	if expectedUpdateCount != updateCount {
		t.Errorf("update ran wrong number of times. expected: %d, got: %d", expectedUpdateCount, updateCount)
	}

	logs, err := store.ListWorkflows(ctx, 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("log length mismatch. expected: %d, got: %d", 1, len(logs))
	}

	got := logs[0]

	expect := &workflow.Workflow{
		Name: "b5/libp2p_node_count",
		Type: workflow.JTDataset,
	}

	if diff := workflow.CompareWorkflows(expect, got); diff != "" {
		t.Errorf("log workflow mismatch (-want +got):\n%s", diff)
	}
}

func TestCronShellScript(t *testing.T) {
	pdci := DefaultCheckInterval
	defer func() { DefaultCheckInterval = pdci }()
	DefaultCheckInterval = time.Millisecond * 50

	updateCount := 0

	wf := &workflow.Workflow{
		ID:        workflow.GenerateWorkflowID(),
		DatasetID: "test/dataset_id",
		Name:      "foo.sh",
		Type:      workflow.JTShellScript,
	}

	// scriptRunner := LocalShellScriptRunner("testdata")
	factory := func(outer context.Context) RunWorkflowFunc {
		return func(ctx context.Context, streams ioes.IOStreams, wf *workflow.Workflow) error {
			switch wf.Type {
			case workflow.JTShellScript:
				updateCount++
				return nil
			}
			t.Fatalf("runner called with invalid workflow: %v", wf)
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	store := workflow.NewMemStore(event.NilBus)
	cron := NewCron(store, factory, event.NilBus)
	if err := cron.Schedule(ctx, wf); err != nil {
		t.Fatal(err)
	}

	if err := cron.Start(ctx); err != nil {
		t.Fatal(err)
	}

	<-ctx.Done()

	expectedUpdateCount := 1
	if expectedUpdateCount != updateCount {
		t.Fatalf("update ran wrong number of times. expected: %d, got: %d", expectedUpdateCount, updateCount)
	}

	logs, err := store.ListWorkflows(ctx, 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Errorf("log length mismatch. expected: %d, got: %d", 1, len(logs))
	}

	got := logs[0]

	expect := &workflow.Workflow{
		ID:        wf.ID,
		DatasetID: wf.DatasetID,
		Name:      "foo.sh",
		Type:      workflow.JTShellScript,
		// RunNumber: 1,
		// RunStart:  got.RunStart,
		// RunStop:   got.RunStop,
	}

	if diff := workflow.CompareWorkflows(expect, got); diff != "" {
		t.Errorf("log workflow mismatch (-want +got):\n%s", diff)
	}
}

func TestRunWorkflow(t *testing.T) {
	ctx := context.Background()

	expectedEvents := []event.Type{
		workflow.ETWorkflowStarted,
		workflow.ETWorkflowCompleted,
	}

	gotEvents := []event.Type{}
	gotEventLock := sync.Mutex{}
	sent := make(chan struct{})
	done := make(chan struct{})

	go func() {
		eventsSent := 0
		for {
			select {
			case <-sent:
				eventsSent++
				if eventsSent == len(expectedEvents) {
					done <- struct{}{}
					return
				}
			case <-time.After(500 * time.Millisecond):
				done <- struct{}{}
			}
		}
	}()

	bus := event.NewBus(ctx)
	handler := func(ctx context.Context, e event.Event) error {
		switch e.Type {
		case workflow.ETWorkflowStarted:
			w, ok := e.Payload.(*workflow.Workflow)
			if !ok {
				t.Fatalf("expected `ETWorkflowStarted` event to emit a *Workflow payload")
			}
			if w.Status != workflow.StatusRunning {
				t.Errorf("expected `ETWorkflowStarted` event to emit a workflow with status 'running', got %q", w.Status)
			}
			if w.LatestStart == nil {
				t.Errorf("expected `ETWorkflowStarted` event to emit a workflow with a `LatestStart`")
			}
			if w.LatestEnd != nil {
				t.Errorf("expected `ETworkflowStarted` event to emit a workflow with no `LatestEnd`, as the workflow is currently running")
			}
			gotEventLock.Lock()
			gotEvents = append(gotEvents, e.Type)
			gotEventLock.Unlock()
			sent <- struct{}{}
		case workflow.ETWorkflowCompleted:
			w, ok := e.Payload.(*workflow.Workflow)
			if !ok {
				t.Fatalf("expected `ETWorkflowCompleted` event to emit a *Workflow payload")
			}
			if w.Status != workflow.StatusSucceeded {
				t.Errorf("expected `ETWorkflowCompleted` event to emit a workflow with status 'succeeded', got %q", w.Status)
			}
			if w.LatestEnd == nil {
				t.Errorf("expected `ETworkflowStarted` event to emit a workflow with a `LatestEnd`")
			}
			gotEventLock.Lock()
			gotEvents = append(gotEvents, e.Type)
			gotEventLock.Unlock()
			sent <- struct{}{}
		default:
			t.Fatalf("unexpected event type: %s", e.Type)
		}
		return nil
	}
	bus.SubscribeTypes(handler, workflow.ETWorkflowStarted, workflow.ETWorkflowCompleted)

	memStore := workflow.NewMemStore(event.NilBus)
	testFunc := func(ctx context.Context, streams ioes.IOStreams, workflow *workflow.Workflow) error {
		return nil
	}
	testFact := func(ctx context.Context) (runner RunWorkflowFunc) {
		return testFunc
	}
	c := NewCron(
		memStore,
		testFact,
		bus,
	)
	wf, err := workflow.NewCronWorkflow("test_workflow", "test_ownerID", "test_datasetID", "R/PT1H")
	if err != nil {
		t.Fatalf("unexpected error making new cron workflow: %s", err)
	}
	c.RunWorkflow(ctx, wf, "")

	<-done
	if diff := cmp.Diff(expectedEvents, gotEvents); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

// requires circular import
func TestDeploy(t *testing.T) {
	tr, err := repotest.NewTempRepo("foo", "deploy_test", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Delete()

	cfg := config.DefaultConfigForTesting()
	cfg.Filesystems = []qfs.Config{
		{Type: "mem"},
		{Type: "local"},
	}
	cfg.Repo.Type = "mem"

	var firedEventWg sync.WaitGroup
	firedEventWg.Add(1)
	handler := func(_ context.Context, e event.Event) error {
		if e.Type == event.ETInstanceConstructed {
			firedEventWg.Done()
		}
		return nil
	}

	// need a mock remote server
	key := lib.InstanceContextKey("RemoteClient")
	ctxV := context.WithValue(context.Background(), key, "mock")
	ctx, cancel := context.WithCancel(ctxV)
	defer cancel()

	// need to create instance with only local resolver
	inst, err := lib.NewInstance(ctx, tr.QriPath, lib.OptConfig(cfg), lib.OptEventHandler(handler, event.ETInstanceConstructed))
	if err != nil {
		t.Fatal(err)
	}
	firedEventWg.Wait()

	store := workflow.NewMemStore(inst.Bus())

	factory := func(ctx context.Context) RunWorkflowFunc {
		return func(ctx context.Context, stream ioes.IOStreams, workflow *workflow.Workflow) error {
			return nil
		}
	}

	// NewService is fro
	c := NewCron(store, factory, inst.Bus())

	username := cfg.Profile.Peername

	workflow, err := workflow.NewCronWorkflow("workflowName", "ownerID", fmt.Sprintf("%s/dataset_bar", username), "R/PT1H")
	if err != nil {
		t.Fatal(err)
	}

	tf := &dataset.Transform{
		Steps: []*dataset.TransformStep{
			{Syntax: "starlark", Category: "setup", Script: ""},
			{Syntax: "starlark", Category: "download", Script: "def download(ctx):\n\treturn"},
			{Syntax: "starlark", Category: "transform", Script: "def transform(ds, ctx):\n\tds.set_body({'first': [1,2,3]})"},
		},
	}

	dp := &DeployParams{
		Apply:     true,
		Workflow:  workflow,
		Transform: tf,
	}
	_, err = c.Deploy(ctx, inst, dp)
	if err != nil {
		t.Fatal(err)
	}
}
