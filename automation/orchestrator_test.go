package automation

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
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

	ran := make(chan string)
	runFuncFactory := func(ctx context.Context) Run {
		return func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow, runID string) error {
			// since we don't actually run anything
			// we need to mock the success of the run
			runStore.Put(&run.State{
				ID:         runID,
				WorkflowID: w.ID,
				Status:     run.RSSucceeded,
			})
			ran <- "ran!"
			return nil
		}
	}
	applied := make(chan string)
	applyFuncFactory := func(ctx context.Context) Apply {
		return func(ctx context.Context, wait bool, runID string, w *workflow.Workflow, ds *dataset.Dataset, secrets map[string]string) error {
			applied <- "applied"
			return nil
		}
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

	cmpOpts := []cmp.Option{
		cmpopts.IgnoreFields(trigger.RuntimeTrigger{}, "id"),
		cmp.AllowUnexported(trigger.RuntimeTrigger{}),
	}
	got, err := o.CreateWorkflow("dataset_id", "profile_id", triggerOpts)
	if err != nil {
		t.Fatal(err)
	}
	expected.ID = got.ID
	if diff := cmp.Diff(expected, got, cmpOpts...); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	if runtimeListener.TriggersExists(expected) {
		t.Fatal("only triggers of active workflows should be added to the runtimeListener")
	}

	got, err = o.GetWorkflow(expected.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expected, got, cmpOpts...); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	expectedWorkflowStartedPayload := &event.WorkflowStartedPayload{
		DatasetID:  got.DatasetID,
		OwnerID:    got.OwnerID,
		WorkflowID: got.WorkflowID(),
	}
	expectedWorkflowStoppedPayload := &event.WorkflowStoppedPayload{
		DatasetID:  got.DatasetID,
		OwnerID:    got.OwnerID,
		WorkflowID: got.WorkflowID(),
		Status:     string(run.RSSucceeded),
	}
	var gotWorkflowStartedPayload *event.WorkflowStartedPayload
	var gotWorkflowStoppedPayload *event.WorkflowStoppedPayload
	workflowEventsHandler := func(ctx context.Context, e event.Event) error {
		ok := true
		switch e.Type {
		case event.ETWorkflowStarted:
			gotWorkflowStartedPayload, ok = e.Payload.(*event.WorkflowStartedPayload)
			if !ok {
				t.Fatal("event.ETWorkflowStarted event should have payload *event.WorkflowStartedPayload")
			}
		case event.ETWorkflowStopped:
			gotWorkflowStoppedPayload, ok = e.Payload.(*event.WorkflowStoppedPayload)
			if !ok {
				t.Fatal("event.ETWorkflowStopped event should have payload *event.WorkflowStoppedPayload")
			}
		}
		return nil
	}

	bus.SubscribeTypes(workflowEventsHandler, event.ETWorkflowStarted, event.ETWorkflowStopped)
	done := errOnTimeout(t, ran, "o.RunWorkflow error: timed out before run function called")
	err = o.RunWorkflow(ctx, got.ID)
	if err != nil {
		t.Fatal(err)
	}
	<-done

	if diff := cmp.Diff(expectedWorkflowStartedPayload, gotWorkflowStartedPayload, cmpopts.IgnoreFields(event.WorkflowStartedPayload{}, "RunID")); diff != "" {
		t.Errorf("WorkflowStartedPayload mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedWorkflowStoppedPayload, gotWorkflowStoppedPayload, cmpopts.IgnoreFields(event.WorkflowStoppedPayload{}, "RunID")); diff != "" {
		t.Errorf("WorkflowStoppedPayload mismatch (-want +got):\n%s", diff)
	}

	done = errOnTimeout(t, applied, "o.ApplyWorkflow error: timed out before apply function called")
	_, err = o.ApplyWorkflow(ctx, false, nil, got, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	<-done

	expected, err = o.DeployWorkflow(expected.ID)
	if err != nil {
		t.Fatalf("DeployWorkflow unexpected error: %s", err)
	}
	// give time for DeployWorkflow to update listeners
	<-time.After(100 * time.Millisecond)
	if runtimeListener.TriggersExists(expected) {
		t.Fatal("orchestrator should not update listeners before the orchestrator has 'Started'.")
	}

	activeTriggers := expected.ActiveTriggers(trigger.RuntimeType)
	if len(activeTriggers) == 0 {
		t.Fatal("workflow unexpectedly has no active triggers")
	}
	wtp := &event.WorkflowTriggerPayload{
		OwnerID:    expected.Owner(),
		WorkflowID: expected.WorkflowID(),
		TriggerID:  activeTriggers[0].ID(),
	}
	done = shouldTimeout(t, ran, "trigger should not trigger a workflow before the orchestrator has run `Start`")
	bus.Publish(ctx, event.ETWorkflowTrigger, wtp)
	<-done

	err = o.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// give time for Start to start each listener
	<-time.After(100 * time.Millisecond)

	if !runtimeListener.TriggersExists(wf) {
		t.Fatalf("Existing workflow triggers for workflow %q must be added to the run store.", wf.ID)
	}
	if !runtimeListener.TriggersExists(expected) {
		t.Fatalf("Existing workflow triggers for workflow %q must be added to the run store.", expected.ID)
	}

	wf, err = o.UndeployWorkflow(wf.ID)
	if err != nil {
		t.Fatalf("Undeployed unexpected error: %s", err)
	}
	// give time for UndeployWorkflow to update listeners
	if runtimeListener.TriggersExists(wf) {
		t.Fatalf("UndeployWorkflow should update listeners")
	}

	done = errOnTimeout(t, ran, "manual trigger error: time out before orchestrator ran a workflow from a trigger")
	runtimeListener.TriggerCh <- wtp
	<-done
	expectedWorkflowStartedPayload = &event.WorkflowStartedPayload{
		DatasetID:  expected.DatasetID,
		OwnerID:    expected.OwnerID,
		WorkflowID: expected.WorkflowID(),
	}
	expectedWorkflowStoppedPayload = &event.WorkflowStoppedPayload{
		DatasetID:  expected.DatasetID,
		OwnerID:    expected.OwnerID,
		WorkflowID: expected.WorkflowID(),
		Status:     string(run.RSSucceeded),
	}

	if diff := cmp.Diff(expectedWorkflowStartedPayload, gotWorkflowStartedPayload, cmpopts.IgnoreFields(event.WorkflowStartedPayload{}, "RunID")); diff != "" {
		t.Errorf("WorkflowStartedPayload mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedWorkflowStoppedPayload, gotWorkflowStoppedPayload, cmpopts.IgnoreFields(event.WorkflowStoppedPayload{}, "RunID")); diff != "" {
		t.Errorf("WorkflowStoppedPayload mismatch (-want +got):\n%s", diff)
	}

	o.Stop()
	done = shouldTimeout(t, ran, "o.Stop error: orchestrator that has stopped listening should not respond to triggers")
	runtimeListener.TriggerCh <- wtp
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

	// We are only interested in ensuring that the runEventsHandler is working
	// properly. To make sure that the mock time only effects the events we
	// are checking for, lets make sure to wait for the `ETWorkflowStarted` event
	// to pass, before we mock the transform events sequence
	workflowStarted := make(chan struct{})
	handleWorkflowStarted := func(ctx context.Context, e event.Event) error {
		if e.Type == event.ETWorkflowStarted {
			workflowStarted <- struct{}{}
		}
		return nil
	}
	bus.SubscribeTypes(handleWorkflowStarted, event.ETWorkflowStarted)
	// this runFunc simulates events emitted by the transform package
	runFuncFactory := func(ctx context.Context) Run {
		return func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow, runID string) error {
			select {
			case <-workflowStarted:
				break
			case <-time.After(200 * time.Millisecond):
				t.Fatal("RunWorkflow error: should have received `ETWorkflowStarted` event")
			}

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
				if len(timestamps) <= eventNumber {
					t.Fatal("NowFunc error, more events than timestamps created")
				}
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
		return func(ctx context.Context, wait bool, runID string, w *workflow.Workflow, ds *dataset.Dataset, secrets map[string]string) error {
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
