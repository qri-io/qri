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
	rttListenTest := trigger.NewEmptyRuntimeTrigger()
	rttListenTest.SetActive(true)
	wf := &workflow.Workflow{
		InitID:   "test_listeners",
		OwnerID:  "profile_id",
		Created:  NowFunc(),
		Triggers: []map[string]interface{}{rttListenTest.ToMap()},
		Active:   true,
	}
	wf, err := workflowStore.Put(ctx, wf)
	if err != nil {
		t.Fatalf("workflowStore.Put unexpected error: %s", err)
	}
	opts := OrchestratorOptions{
		WorkflowStore: workflowStore,
		RunStore:      runStore,
		Listeners:     []trigger.Listener{runtimeListener},
	}

	applied := make(chan string)
	runner := newTestWorkflowRunner(runStore, applied)

	o, err := NewOrchestrator(ctx, bus, runner, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer o.Stop()

	prevTriggerNewID := trigger.NewID
	defer func() {
		trigger.NewID = prevTriggerNewID
	}()
	triggerIDIndex := 0
	triggerIDs := [2]string{
		"id1",
		"id2",
	}
	trigger.NewID = func() string {
		if triggerIDIndex >= len(triggerIDs) {
			t.Fatal("trigger.NewID called more times then expected")
		}
		id := triggerIDs[triggerIDIndex]
		triggerIDIndex++
		return id
	}
	expected := &workflow.Workflow{
		InitID:  "dataset_id",
		OwnerID: "profile_id",
		Created: NowFunc(),
		Triggers: []map[string]interface{}{
			map[string]interface{}{
				"id":           triggerIDs[0],
				"type":         trigger.RuntimeType,
				"active":       false,
				"advanceCount": 0,
			},
			map[string]interface{}{
				"id":           triggerIDs[1],
				"type":         trigger.RuntimeType,
				"active":       true,
				"advanceCount": 0,
			}},
	}

	got, err := o.SaveWorkflow(ctx, &workflow.Workflow{
		InitID:  "dataset_id",
		OwnerID: "profile_id",
		Triggers: []map[string]interface{}{
			map[string]interface{}{"type": trigger.RuntimeType},
			map[string]interface{}{"type": trigger.RuntimeType, "active": true},
		}})
	if err != nil {
		t.Fatal(err)
	}
	expected.ID = got.ID
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	if runtimeListener.TriggersExists(expected) {
		t.Fatal("only triggers of active workflows should be added to the runtimeListener")
	}

	got, err = o.GetWorkflow(ctx, expected.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	runID := "runID_1"
	expectedWorkflowEvents := []interface{}{
		event.WorkflowStartedEvent{
			InitID:     got.InitID,
			OwnerID:    got.OwnerID,
			WorkflowID: got.WorkflowID(),
			RunID:      runID,
		},
		event.WorkflowStoppedEvent{
			InitID:     got.InitID,
			OwnerID:    got.OwnerID,
			WorkflowID: got.WorkflowID(),
			RunID:      runID,
			Status:     string(run.RSSucceeded),
		},
	}
	gotWorkflowEvents := []interface{}{}
	workflowStoppedEventFired := make(chan string)
	workflowEventsHandler := func(ctx context.Context, e event.Event) error {
		switch e.Type {
		case event.ETAutomationWorkflowStarted:
			gotWorkflowStartedEvent, ok := e.Payload.(event.WorkflowStartedEvent)
			if !ok {
				t.Fatal("event.ETAutomationWorkflowStarted event should have payload event.WorkflowStartedEvent")
			}
			gotWorkflowEvents = append(gotWorkflowEvents, gotWorkflowStartedEvent)
		case event.ETAutomationWorkflowStopped:
			gotWorkflowStoppedEvent, ok := e.Payload.(event.WorkflowStoppedEvent)
			if !ok {
				t.Fatal("event.ETAutomationWorkflowStopped event should have payload event.WorkflowStoppedEvent")
			}
			gotWorkflowEvents = append(gotWorkflowEvents, gotWorkflowStoppedEvent)
			workflowStoppedEventFired <- "workflow finished"
		}
		return nil
	}

	bus.SubscribeTypes(workflowEventsHandler, event.ETAutomationWorkflowStarted, event.ETAutomationWorkflowStopped)
	done := errOnTimeout(t, workflowStoppedEventFired, "o.RunWorkflow error: timed out before `ETAutomationWorkflowStopped` event fired")
	_, err = o.RunWorkflow(ctx, got.ID, runID)
	if err != nil {
		t.Fatal(err)
	}
	<-done

	if diff := cmp.Diff(expectedWorkflowEvents, gotWorkflowEvents); diff != "" {
		t.Errorf("Workflow events mismatch (-want +got):\n%s", diff)
	}
	gotWorkflowEvents = []interface{}{}

	done = errOnTimeout(t, applied, "o.ApplyWorkflow error: timed out before apply function called")
	_, err = o.ApplyWorkflow(ctx, false, nil, got, nil, WorkflowRunParams{})
	if err != nil {
		t.Fatal(err)
	}
	<-done

	expected.Active = true
	expected, err = o.SaveWorkflow(ctx, expected)
	if err != nil {
		t.Fatalf("SaveWorkflow unexpected error: %s", err)
	}
	// give time for SaveWorkflow to update listeners
	<-time.After(100 * time.Millisecond)
	if runtimeListener.TriggersExists(expected) {
		t.Fatal("orchestrator should not update listeners before the orchestrator has 'Started'.")
	}

	activeTriggers := expected.ActiveTriggers(trigger.RuntimeType)
	if len(activeTriggers) == 0 {
		t.Fatal("workflow unexpectedly has no active triggers")
	}
	triggerID := activeTriggers[0]["id"].(string)
	wtp := event.WorkflowTriggerEvent{
		OwnerID:    expected.Owner(),
		WorkflowID: expected.WorkflowID(),
		TriggerID:  triggerID,
	}
	done = shouldTimeout(t, workflowStoppedEventFired, "trigger should not trigger a workflow before the orchestrator has run `Start`")
	bus.Publish(ctx, event.ETAutomationWorkflowTrigger, wtp)
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

	wf.Active = false
	wf, err = o.SaveWorkflow(ctx, wf)
	if err != nil {
		t.Fatalf("SaveWorkflow unexpected error: %s", err)
	}
	// give time for SaveWorkflow to update listeners
	if runtimeListener.TriggersExists(wf) {
		t.Fatalf("SaveWorkflow should update listeners")
	}
	expectedWorkflowEvents = []interface{}{
		event.WorkflowStartedEvent{
			InitID:     expected.InitID,
			OwnerID:    expected.OwnerID,
			WorkflowID: expected.WorkflowID(),
		},
		event.WorkflowStoppedEvent{
			InitID:     expected.InitID,
			OwnerID:    expected.OwnerID,
			WorkflowID: expected.WorkflowID(),
			Status:     string(run.RSSucceeded),
		},
	}

	done = errOnTimeout(t, workflowStoppedEventFired, "manual trigger error: time out before orchestrator published the `ETAutomationWorkflowStopped` event")
	runtimeListener.TriggerCh <- wtp
	<-done

	if diff := cmp.Diff(expectedWorkflowEvents, gotWorkflowEvents, cmpopts.IgnoreFields(event.WorkflowStartedEvent{}, "RunID"), cmpopts.IgnoreFields(event.WorkflowStoppedEvent{}, "RunID")); diff != "" {
		t.Errorf("Workflow events mismatch (-want +got):\n%s", diff)
	}

	o.Stop()
	done = shouldTimeout(t, workflowStoppedEventFired, "o.Stop error: orchestrator that has stopped listening should not respond to triggers")
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
	wf, err := workflowStore.Put(ctx, &workflow.Workflow{
		InitID:  "dataset_id",
		OwnerID: "owner_id",
		Created: &time.Time{},
	})
	if err != nil {
		t.Fatal(err)
	}

	// We are only interested in ensuring that the runEventsHandler is working
	// properly. To make sure that the mock time only effects the events we
	// are checking for, lets make sure to wait for the `ETAutomationWorkflowStarted` event
	// to pass, before we mock the transform events sequence
	transformStopped := make(chan struct{})
	handleEventTiming := func(ctx context.Context, e event.Event) error {
		if e.Type == event.ETTransformStop {
			transformStopped <- struct{}{}
		}
		return nil
	}
	bus.SubscribeTypes(handleEventTiming, event.ETTransformStop)
	runID := "TestRunStoreEventsRunID"

	// event 0
	r0 := &run.State{
		ID:         runID,
		WorkflowID: wf.ID,
		Status:     run.RSRunning,
	}

	// event 1
	expectedStartStep := &run.StepState{
		Name:     "step start",
		Category: "step start category",
		Status:   run.RSRunning,
	}
	r1 := r0.Copy()
	r1.Steps = []*run.StepState{expectedStartStep}

	// event 2
	expectedPrintEvent := event.Event{
		Type:      event.ETTransformPrint,
		SessionID: runID,
		Payload: event.TransformMessage{
			Msg: "transform print",
		},
	}
	r2 := r1.Copy()
	r2.Steps = []*run.StepState{expectedStartStep.Copy()}
	r2.Steps[0].Output = []event.Event{expectedPrintEvent}

	// event 3
	expectedErrorEvent := event.Event{
		Type:      event.ETTransformError,
		SessionID: runID,
		Payload: event.TransformMessage{
			Msg: "transform error",
		},
	}
	r3 := r2.Copy()
	r3.Steps = []*run.StepState{expectedStartStep.Copy()}
	r3.Steps[0].Output = []event.Event{expectedPrintEvent, expectedErrorEvent}

	// event 4
	expectedDatasetPreviewEvent := event.Event{
		Type:      event.ETTransformDatasetPreview,
		SessionID: runID,
		Payload:   "transform dataset preview",
	}
	r4 := r3.Copy()
	r4.Steps = []*run.StepState{expectedStartStep.Copy()}
	r4.Steps[0].Output = []event.Event{expectedPrintEvent, expectedErrorEvent, expectedDatasetPreviewEvent}

	// event 5
	expectedStartStepDone := expectedStartStep.Copy()
	expectedStartStepDone.Status = run.RSSucceeded
	r5 := r4.Copy()
	r5.Steps = []*run.StepState{expectedStartStepDone}
	r5.Steps[0].Output = []event.Event{expectedPrintEvent, expectedErrorEvent, expectedDatasetPreviewEvent}

	// event 6
	expectedSkipStep := &run.StepState{
		Name:     "step skip",
		Category: "step skip category",
		Status:   run.RSSkipped,
	}
	r6 := r5.Copy()
	r6.Steps = []*run.StepState{expectedStartStepDone, expectedSkipStep}
	r6.Steps[0].Output = []event.Event{expectedPrintEvent, expectedErrorEvent, expectedDatasetPreviewEvent}

	// event 7
	r7 := r6.Copy()
	r7.Status = run.RSSucceeded

	events := []simulatedRunEvent{
		{
			r0, event.ETTransformStart, nil,
		},
		{
			r1, event.ETTransformStepStart, event.TransformStepLifecycle{
				Name:     "step start",
				Category: "step start category",
			},
		},
		{
			r2, event.ETTransformPrint, event.TransformMessage{Msg: "transform print"},
		},
		{
			r3, event.ETTransformError, event.TransformMessage{Msg: "transform error"},
		},
		{
			r4, event.ETTransformDatasetPreview, "transform dataset preview",
		},
		{
			r5, event.ETTransformStepStop, event.TransformStepLifecycle{Status: "succeeded"},
		},
		{
			r6, event.ETTransformStepSkip, event.TransformStepLifecycle{Name: "step skip", Category: "step skip category", Status: "skipped"},
		},
		{
			r7, event.ETTransformStop, event.TransformLifecycle{Status: "succeeded"},
		},
	}
	simulator := newWorkflowRunSimulator(t, runStore, bus, events)

	opts := OrchestratorOptions{
		WorkflowStore: workflowStore,
		RunStore:      runStore,
		Listeners:     []trigger.Listener{listener},
	}
	o, err := NewOrchestrator(ctx, bus, simulator, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer o.Stop()
	if _, err := o.RunWorkflow(ctx, wf.ID, runID); err != nil {
		t.Fatal(err)
	}
	<-transformStopped
}

func confirmStoredRun(ctx context.Context, t *testing.T, s run.Store, expect *run.State) {
	t.Helper()
	got, err := s.Get(ctx, expect.ID)
	if err != nil {
		t.Fatalf("getting run: %s", err)
	}

	if diff := cmp.Diff(expect, got, cmpopts.IgnoreFields(run.State{}, "StartTime", "StopTime", "Duration"), cmpopts.IgnoreFields(run.StepState{}, "StartTime", "StopTime", "Duration"), cmpopts.IgnoreFields(event.Event{}, "Timestamp")); diff != "" {
		t.Errorf("stored run mismatch: (-want +got):\n%s", diff)
	}
}

// a workflow runner that doesn't do too much, just for testing
type testWorkflowRunner struct {
	store   run.Store
	applied chan string
}

func newTestWorkflowRunner(store run.Store, applied chan string) *testWorkflowRunner {
	return &testWorkflowRunner{
		store:   store,
		applied: applied,
	}
}

func (r *testWorkflowRunner) RunEphemeral(ctx context.Context, runID string, wf *workflow.Workflow, ds *dataset.Dataset, wait bool, params WorkflowRunParams) error {
	r.applied <- "applied"
	return nil
}

func (r *testWorkflowRunner) RunAndCommit(ctx context.Context, runID string, wf *workflow.Workflow, streams ioes.IOStreams, params WorkflowRunParams) error {
	//return r.owner.run(ctx, streams, wf, runID)
	// since we don't actually run anything
	// we need to mock the success of the run
	r.store.Put(ctx, &run.State{
		ID:         runID,
		WorkflowID: wf.ID,
		Status:     run.RSSucceeded,
	})
	<-time.After(50 * time.Millisecond)
	return nil
}

// a simulated event and run state
type simulatedRunEvent struct {
	state   *run.State
	etype   event.Type
	payload interface{}
}

// a workflow runner that simulates events
type workflowRunSimulator struct {
	t      *testing.T
	bus    event.Bus
	store  run.Store
	events []simulatedRunEvent
}

func newWorkflowRunSimulator(t *testing.T, store run.Store, bus event.Bus, events []simulatedRunEvent) *workflowRunSimulator {
	return &workflowRunSimulator{
		t:      t,
		bus:    bus,
		store:  store,
		events: events,
	}
}

// RunAndCommit simulates events emitted by the transform package
func (r *workflowRunSimulator) RunAndCommit(ctx context.Context, runID string, wf *workflow.Workflow, streams ioes.IOStreams, params WorkflowRunParams) error {
	for _, runEvent := range r.events {
		r.bus.PublishID(ctx, runEvent.etype, runID, runEvent.payload)
		confirmStoredRun(ctx, r.t, r.store, runEvent.state)
	}
	return nil
}

func (r *workflowRunSimulator) RunEphemeral(ctx context.Context, runID string, wf *workflow.Workflow, ds *dataset.Dataset, wait bool, params WorkflowRunParams) error {
	return nil
}
