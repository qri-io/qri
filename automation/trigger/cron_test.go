package trigger_test

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/qri/automation/spec"
	"github.com/qri-io/qri/automation/trigger"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/event"
)

func TestCronTrigger(t *testing.T) {
	opts := map[string]interface{}{
		"type":        trigger.CronType,
		"id":          "test_1",
		"active":      true,
		"periodicity": "R/2021-07-13T21:15:00.000Z/P1H",
	}
	ct, err := trigger.NewCronTrigger(opts)
	if err != nil {
		t.Fatal(err)
	}
	adv := map[string]interface{}{
		"type":         trigger.CronType,
		"id":           "test_1",
		"active":       true,
		"periodicity":  "R/2021-07-13T22:15:00Z/P1H",
		"nextRunStart": "2021-07-13T22:15:00Z",
	}
	spec.AssertTrigger(t, ct, adv)
}

func TestCronListener(t *testing.T) {
	wf := &workflow.Workflow{
		ID:      "test_workflow_id",
		OwnerID: "test Owner id",
		Active:  true,
		Triggers: []map[string]interface{}{
			map[string]interface{}{
				"id":          "trigger1",
				"active":      true,
				"type":        trigger.CronType,
				"periodicity": "R/2021-07-13T11:30:00.000Z/P1H",
			},
		},
	}
	prevNowFunc := trigger.NowFunc
	defer func() {
		trigger.NowFunc = prevNowFunc
	}()
	timeIsNoon := func() time.Time {
		ti, err := time.Parse(time.RFC3339, "2021-07-13T12:00:00.000Z")
		if err != nil {
			t.Fatal("error parsing time midnight")
		}
		return ti
	}
	listenerConstructor := func(ctx context.Context, bus event.Bus) (trigger.Listener, func(), func()) {
		trigger.NowFunc = timeIsNoon
		cl := trigger.NewCronListenerInterval(bus, time.Millisecond*100)
		if err := cl.Listen(wf); err != nil {
			t.Fatalf("CronListener.Listen error, %s", err)
		}
		activateTrigger := func() {}
		advanceTrigger := func() {
			wf.Triggers[0]["periodicity"] = "R/2021-07-13T12:30:00.000Z/P1H"
			if err := cl.Listen(wf); err != nil {
				t.Fatalf("CronListener.Listen error, %s", err)
			}
		}
		return cl, activateTrigger, advanceTrigger
	}
	spec.AssertListener(t, listenerConstructor)
}
