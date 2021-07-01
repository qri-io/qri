package trigger_test

import (
	"context"
	"testing"

	"github.com/qri-io/qri/automation/spec"
	"github.com/qri-io/qri/automation/trigger"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
)

func TestRuntimeTrigger(t *testing.T) {
	rt := trigger.NewRuntimeTrigger()
	spec.AssertTrigger(t, rt)
}

func TestRuntimeListener(t *testing.T) {
	wf := &workflow.Workflow{
		ID:       workflow.ID("test workflow id"),
		OwnerID:  "test Owner id",
		Deployed: true,
		Triggers: []trigger.Trigger{},
	}
	listenerConstructor := func(ctx context.Context, bus event.Bus) (trigger.Listener, func()) {
		rl := trigger.NewRuntimeListener(ctx, bus)
		triggerOpts := map[string]interface{}{
			"active": true,
			"type":   trigger.RuntimeType,
		}

		trig, err := rl.ConstructTrigger(triggerOpts)
		if err != nil {
			t.Fatalf("RuntimeListener.ConstructTrigger unexpected error: %s", err)
		}
		rt, ok := trig.(*trigger.RuntimeTrigger)
		if !ok {
			t.Fatal("RuntimeListener.ConstructTrigger did not return a RuntimeTrigger")
		}
		activateTrigger := func() {
			if rl.TriggerCh == nil {
				return
			}
			wtp := &event.WorkflowTriggerPayload{
				OwnerID:    wf.OwnerID,
				WorkflowID: wf.ID.String(),
				TriggerID:  rt.ID(),
			}
			rl.TriggerCh <- wtp
		}

		wf.Triggers = []trigger.Trigger{rt}
		if err := rl.Listen(wf); err != nil {
			t.Fatalf("RuntimeListener.Listen unexpected error: %s", err)
		}
		return rl, activateTrigger
	}
	spec.AssertListener(t, listenerConstructor)

	ctx := context.Background()
	l, _ := listenerConstructor(ctx, event.NilBus)
	rl, ok := l.(*trigger.RuntimeListener)
	if !ok {
		t.Fatal("RuntimeListener unexpected assertion error, listenerConstructor should return a runtimeListener")
	}
	wf.Triggers = []trigger.Trigger{}
	if err := rl.Listen(wf); err != nil {
		t.Fatalf("RuntimeListener.Listen unexpected error: %s", err)
	}
	if rl.TriggersExists(wf) {
		t.Errorf("RuntimeListener.Listen error: should remove triggers from its internal store when given an updated workflow with a no longer active trigger")
	}
}

func TestRuntimeListenerListen(t *testing.T) {
	ctx := context.Background()
	bus := event.NewBus(ctx)
	rl := trigger.NewRuntimeListener(ctx, bus)

	aID := profile.ID("a")
	wfA1 := &workflow.Workflow{
		OwnerID:  aID,
		ID:       workflow.ID("workflow 1"),
		Deployed: true,
	}
	bID := profile.ID("b")
	wfB1 := &workflow.Workflow{
		OwnerID:  bID,
		ID:       workflow.ID("workflow 1"),
		Deployed: true,
	}
	if err := rl.Listen([]trigger.Source{wfA1, wfB1}...); err != nil {
		t.Fatal(err)
	}
	if rl.TriggersExists(wfA1) || rl.TriggersExists(wfB1) {
		t.Fatal("workflow with no triggers should not exist in the Listener")
	}
	trig1 := trigger.NewRuntimeTrigger()
	trig2 := trigger.NewRuntimeTrigger()
	wfA1.Triggers = []trigger.Trigger{trig1, trig2}
	if err := rl.Listen([]trigger.Source{wfA1}...); err != nil {
		t.Fatal(err)
	}
	if rl.TriggersExists(wfA1) {
		t.Fatal("workflow with no active triggers should not exist in the Listener")
	}
	trig1.SetActive(true)
	if err := rl.Listen([]trigger.Source{wfA1}...); err != nil {
		t.Fatal(err)
	}
	if !rl.TriggersExists(wfA1) {
		t.Fatal("workflow with an active trigger should exist in the listener")
	}
	trig2.SetActive(true)

	if rl.TriggersExists(wfA1) {
		t.Fatal("workflow with non matching trigger list should not exist in the listener")
	}

	if err := rl.Listen([]trigger.Source{wfA1}...); err != nil {
		t.Fatal(err)
	}
	if !rl.TriggersExists(wfA1) {
		t.Fatal("Listen did not update triggers for wfA1")
	}

	wfA2 := &workflow.Workflow{
		OwnerID:  aID,
		ID:       workflow.ID("workflow 2"),
		Triggers: []trigger.Trigger{trig1, trig2},
		Deployed: true,
	}

	wfB1.Triggers = []trigger.Trigger{trig1}
	if err := rl.Listen([]trigger.Source{wfB1, wfA2}...); err != nil {
		t.Fatal(err)
	}
	if !rl.TriggersExists(wfA2) {
		t.Fatal("Listen did not add wfA2")
	}
	if !rl.TriggersExists(wfA1) {
		t.Fatal("Listen erroneously removed wfA1")
	}
	if !rl.TriggersExists(wfB1) {
		t.Fatal("Listen did not add wfB1")
	}
	wfA1.Triggers = []trigger.Trigger{}
	if err := rl.Listen([]trigger.Source{wfA1}...); err != nil {
		t.Fatal(err)
	}
	if rl.TriggersExists(wfA1) {
		t.Fatal("Listen did not remove wfA1 when wfA1 had no triggers")
	}
}
