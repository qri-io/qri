package automation

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/event"
)

func TestIntegration(t *testing.T) {
	// mock time
	prevNow := NowFunc
	defer func() {
		NowFunc = prevNow
	}()
	now := time.Now()
	NowFunc = func() *time.Time { return &now }

	ctx := context.Background()
	ran := make(chan string)
	runFuncFactory := func(ctx context.Context) Run {
		return func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow, runID string) error {
			ran <- "ran!"
			return nil
		}
	}
	applied := make(chan string)
	applyFuncFactory := func(ctx context.Context) Apply {
		return func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow) error {
			applied <- "applied"
			return nil
		}
	}

	bus := event.NewBus(ctx)

	runStore := run.NewMemStore(bus)
	workflowStore := workflow.NewMemStore()
	opts := OrchestratorOptions{
		WorkflowStore: workflowStore,
		RunStore:      runStore,
	}
	o, err := NewOrchestrator(ctx, bus, runFuncFactory, applyFuncFactory, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer o.Shutdown()

	expected := &workflow.Workflow{
		DatasetID: "dataset_id",
		OwnerID:   "profile_id",
		Created:   NowFunc(),
	}

	got, err := o.CreateWorkflow("dataset_id", "profile_id")
	if err != nil {
		t.Fatal(err)
	}
	expected.ID = got.ID
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	got, err = o.GetWorkflow(expected.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	done := errOnTimeout(t, ran, "o.RunWorkflow error: timed out before run function called")
	err = o.RunWorkflow(ctx, got.ID)
	if err != nil {
		t.Fatal(err)
	}
	<-done

	done = errOnTimeout(t, applied, "o.ApplyWorkflow error: timed out before apply function called")
	err = o.ApplyWorkflow(ctx, got.ID)
	if err != nil {
		t.Fatal(err)
	}
	<-done

	done = errOnTimeout(t, ran, "o.handleTrigger error: time out before run function called")
	bus.Publish(ctx, event.ETWorkflowTrigger, expected.ID)
	<-done
}

func errOnTimeout(t *testing.T, c chan string, errMsg string) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		select {
		case msg := <-c:
			t.Log(msg)
		case <-time.After(200 * time.Millisecond):
			t.Errorf(errMsg)
		}
		done <- struct{}{}
	}()
	return done
}
