package trigger_test

import (
	"context"
	"testing"

	"github.com/qri-io/qri/automation/spec"
	"github.com/qri-io/qri/automation/trigger"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/event"
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
			rt.Trigger(rl.TriggerCh, "test workflow id")
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
	if rl.TriggerExists(wf) {
		t.Errorf("RuntimeListener.Listen error: should remove triggers from its internal store when given an updated workflow with a no longer active trigger")
	}
}
