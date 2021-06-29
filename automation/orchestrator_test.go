package automation

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/trigger"
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

	runStore := run.NewMemStore()
	workflowStore := workflow.NewMemStore()
	runtimeListener := trigger.NewRuntimeListener(ctx, bus)
	rttListenTest := trigger.NewRuntimeTrigger()
	rttListenTest.SetActive(true)
	wf := &workflow.Workflow{
		DatasetID: "test_listeners",
		OwnerID:   "profile_id",
		Created:   NowFunc(),
		Triggers:  []trigger.Trigger{rttListenTest},
		Deployed:  true,
	}
	wf, err := workflowStore.Put(wf)
	if err != nil {
		t.Fatalf("workflowStore.Put unexpected error: %s", err)
	}
	opts := OrchestratorOptions{
		WorkflowStore: workflowStore,
		RunStore:      runStore,
		Listeners:     []trigger.Listener{runtimeListener},
	}
	o, err := NewOrchestrator(ctx, bus, runFuncFactory, applyFuncFactory, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer o.Shutdown()

	rtt1 := trigger.NewRuntimeTrigger()
	rtt2 := trigger.NewRuntimeTrigger()
	rtt2.SetActive(true)
	expected := &workflow.Workflow{
		DatasetID: "dataset_id",
		OwnerID:   "profile_id",
		Created:   NowFunc(),
		Triggers:  []trigger.Trigger{rtt1, rtt2},
	}

	triggerOpts := []map[string]interface{}{
		map[string]interface{}{"type": trigger.RuntimeType},
		map[string]interface{}{"type": trigger.RuntimeType, "active": true},
	}

	allowUnexported := cmp.AllowUnexported(trigger.RuntimeTrigger{})
	got, err := o.CreateWorkflow("dataset_id", "profile_id", triggerOpts)
	if err != nil {
		t.Fatal(err)
	}
	expected.ID = got.ID
	if diff := cmp.Diff(expected, got, allowUnexported); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	if runtimeListener.TriggerExists(expected) {
		t.Fatal("only triggers of active workflows should be added to the runtimeListener")
	}

	got, err = o.GetWorkflow(expected.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expected, got, allowUnexported); diff != "" {
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

	if err := o.DeployWorkflow(expected.ID); err != nil {
		t.Fatalf("DeployWorkflow unexpected error: %s", err)
	}
	// give time for DeployWorkflow to update listeners
	<-time.After(100 * time.Millisecond)
	if !runtimeListener.TriggerExists(expected) {
		t.Fatal("orchestrator must update the listeners when the workflow status changes")
	}

	done = errOnTimeout(t, ran, "o.handleTrigger error: time out before run function called")
	bus.Publish(ctx, event.ETWorkflowTrigger, expected.WorkflowID())
	<-done

	err = o.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// give time for Start to start each listener
	<-time.After(100 * time.Millisecond)

	if !runtimeListener.TriggerExists(wf) {
		t.Fatal("Existing workflow triggers must be added to the run store.")
	}

	done = errOnTimeout(t, ran, "manual trigger error: time out before orchestrator ran a workflow from a trigger")
	rtt2.Trigger(runtimeListener.TriggerCh, expected.WorkflowID())
	<-done

	o.Stop()
	done = shouldTimeout(t, ran, "o.Stop error: orchestrator that has stopped listening should not respond to triggers")
	rtt2.Trigger(runtimeListener.TriggerCh, expected.WorkflowID())
	<-done
}

func errOnTimeout(t *testing.T, c chan string, errMsg string) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		select {
		case msg := <-c:
			t.Log(msg)
		case <-time.After(200 * time.Millisecond):
			t.Error(errMsg)
		}
		done <- struct{}{}
	}()
	return done
}

func shouldTimeout(t *testing.T, c chan string, errMsg string) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		select {
		case <-c:
			t.Error(errMsg)
		case <-time.After(200 * time.Millisecond):
			t.Log("expected timeout")
		}
		done <- struct{}{}
	}()
	return done
}

func TestRunStoreEvents(t *testing.T) {
	// mock time
	prevNow := event.NowFunc
	defer func() { event.NowFunc = prevNow }()

	timestamps := []time.Time{}
	totalEventsEmitted := 8
	eventNumber := 0
	for i := 0; i < totalEventsEmitted; i++ {
		t := time.Unix(int64(i), 0)
		timestamps = append(timestamps, t)
	}
	event.NowFunc = func() time.Time {
		t := timestamps[eventNumber]
		eventNumber++
		return t
	}

	timestampNum := 0
	nextTimestamp := func() *time.Time {
		if timestampNum >= len(timestamps) {
			t.Fatal("timestamp error, out of bounds")
		}
		t := timestamps[timestampNum]
		timestampNum++
		return &t
	}

	ctx := context.Background()
	bus := event.NewBus(ctx)
	listener := trigger.NewRuntimeListener(ctx, bus)
	runStore := run.NewMemStore()
	workflowStore := workflow.NewMemStore()
	wf, err := workflowStore.Put(&workflow.Workflow{
		DatasetID: "dataset_id",
		OwnerID:   "owner_id",
		Created:   &time.Time{},
	})
	if err != nil {
		t.Fatal(err)
	}
	r := &run.State{WorkflowID: wf.ID}

	// this runFunc simulates events emitted by the transform package
	runFuncFactory := func(ctx context.Context) Run {
		return func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow, runID string) error {
			r.ID = runID

			// event 0
			bus.PublishID(ctx, event.ETTransformStart, r.ID, nil)
			ts := nextTimestamp()
			r.Status = run.RSRunning
			r.StartTime = ts
			confirmStoredRun(t, runStore, r)

			// event 1
			bus.PublishID(ctx, event.ETTransformStepStart, r.ID, event.TransformStepLifecycle{
				Name:     "step start",
				Category: "step start category",
			})
			ts = nextTimestamp()
			expectedStep := &run.StepState{
				Name:      "step start",
				Category:  "step start category",
				Status:    run.RSRunning,
				StartTime: ts,
			}
			r.Steps = append(r.Steps, expectedStep)
			confirmStoredRun(t, runStore, r)

			// event 2
			bus.PublishID(ctx, event.ETTransformPrint, r.ID, "transform print")
			ts = nextTimestamp()
			expectedPrintEvent := event.Event{
				Type:      event.ETTransformPrint,
				Timestamp: (*ts).UnixNano(),
				SessionID: r.ID,
				Payload:   "transform print",
			}
			r.Steps[0].Output = []event.Event{expectedPrintEvent}
			confirmStoredRun(t, runStore, r)

			// event 3
			bus.PublishID(ctx, event.ETTransformError, r.ID, "transform error")
			ts = nextTimestamp()
			expectedErrorEvent := event.Event{
				Type:      event.ETTransformError,
				Timestamp: (*ts).UnixNano(),
				SessionID: r.ID,
				Payload:   "transform error",
			}
			r.Steps[0].Output = []event.Event{expectedPrintEvent, expectedErrorEvent}
			confirmStoredRun(t, runStore, r)

			// event 4
			bus.PublishID(ctx, event.ETTransformDatasetPreview, r.ID, "transform dataset preview")
			ts = nextTimestamp()
			expectedDatasetPreviewEvent := event.Event{
				Type:      event.ETTransformDatasetPreview,
				Timestamp: (*ts).UnixNano(),
				SessionID: r.ID,
				Payload:   "transform dataset preview",
			}
			r.Steps[0].Output = []event.Event{expectedPrintEvent, expectedErrorEvent, expectedDatasetPreviewEvent}
			confirmStoredRun(t, runStore, r)

			// event 5
			bus.PublishID(ctx, event.ETTransformStepStop, r.ID, event.TransformStepLifecycle{Status: "succeeded"})
			ts = nextTimestamp()
			expectedStep = r.Steps[0]
			expectedStep.StopTime = ts
			expectedStep.Status = run.RSSucceeded
			expectedStep.Duration = int(expectedStep.StopTime.Sub(*expectedStep.StartTime))
			confirmStoredRun(t, runStore, r)

			// event 6
			bus.PublishID(ctx, event.ETTransformStepSkip, r.ID, event.TransformStepLifecycle{Name: "step skip", Category: "step skip category", Status: "skipped"})
			ts = nextTimestamp()
			expectedSkipStep := &run.StepState{
				Name:     "step skip",
				Category: "step skip category",
				Status:   run.RSSkipped,
			}
			r.Steps = append(r.Steps, expectedSkipStep)
			confirmStoredRun(t, runStore, r)

			// event 7
			bus.PublishID(ctx, event.ETTransformStop, r.ID, event.TransformLifecycle{Status: "succeeded"})
			ts = nextTimestamp()
			r.StopTime = ts
			r.Status = run.RSSucceeded
			r.Duration = int(r.StopTime.Sub(*r.StartTime))
			confirmStoredRun(t, runStore, r)

			return nil
		}
	}
	applyFuncFactory := func(ctx context.Context) Apply {
		return func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow) error {
			return nil
		}
	}

	opts := OrchestratorOptions{
		WorkflowStore: workflowStore,
		RunStore:      runStore,
		Listeners:     []trigger.Listener{listener},
	}
	o, err := NewOrchestrator(ctx, bus, runFuncFactory, applyFuncFactory, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer o.Shutdown()

	if err := o.RunWorkflow(ctx, wf.ID); err != nil {
		t.Fatal(err)
	}
}

func confirmStoredRun(t *testing.T, s run.Store, expect *run.State) {
	t.Helper()
	got, err := s.Get(expect.ID)
	if err != nil {
		t.Fatalf("getting run: %s", err)
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("stored run mismatch: (-want +got):\n%s", diff)
	}
}
