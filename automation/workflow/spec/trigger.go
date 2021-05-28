package spec

import (
	"encoding/json"
	"testing"

	"github.com/qri-io/qri/automation/workflow"
)

func AssertTrigger(t *testing.T, trigger workflow.Trigger) {
	if trigger.Type() == "" {
		t.Error("Type method must return a non-empty TriggerType")
	}
	if err := trigger.SetEnabled(true); err != nil {
		t.Fatalf("trigger.SetEnabled unexpected error: %s", err)
	}
	if !trigger.Enabled() {
		t.Error("expected trigger.Enabled() to be true after trigger.SetEnabled(true)")
	}
	if err := trigger.SetEnabled(false); err != nil {
		t.Fatalf("trigger.SetEnabled unexpected error: %s", err)
	}
	if trigger.Enabled() {
		t.Error("expected trigger.Enabled() to be false after trigger.SetEnabled(false)")
	}
	triggerBytes, err := json.Marshal(trigger)
	if err != nil {
		t.Fatalf("json.Marshal unexpected error: %s", err)
	}
	triggerObj := map[string]interface{}{}
	if err := json.Unmarshal(triggerBytes, &triggerObj); err != nil {
		t.Fatalf("json.Unmarshal unexpected error: %s", err)
	}
	triggerType, ok := triggerObj["type"]
	if !ok {
		t.Fatal("json.Marshal error, expected Type field to exist")
	}
	if triggerType != trigger.Type().String() {
		t.Fatalf("json.Marshal error, expected marshalled type %q to match trigger.Type() %q", triggerType, trigger.Type())
	}
	triggerObj["type"] = "assert test trigger type"
	triggerBytes, err = json.Marshal(triggerObj)
	if err != nil {
		t.Fatalf("json.Unmarshal unexpected error: %s", err)
	}
	if err := json.Unmarshal(triggerBytes, trigger); err != nil {
		if err != nil {
			t.Fatalf("json.Unmarshal unexpected error: %s", err)
		}
	}
	if triggerObj["type"] != trigger.Type().String() {
		t.Fatalf("json.Unmarshal error, expected unmarshaled type %s to match %s", trigger.Type(), triggerObj["type"])
	}
}
