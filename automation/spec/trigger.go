package spec

import (
	"encoding/json"
	"github.com/qri-io/qri/automation/trigger"
	"testing"
)

func AssertTrigger(t *testing.T, trig trigger.Trigger) {
	if trig.Type() == "" {
		t.Error("Type method must return a non-empty TriggerType")
	}
	if err := trig.SetEnabled(true); err != nil {
		t.Fatalf("trigger.SetEnabled unexpected error: %s", err)
	}
	if !trig.Enabled() {
		t.Error("expected trigger.Enabled() to be true after trigger.SetEnabled(true)")
	}
	if err := trig.SetEnabled(false); err != nil {
		t.Fatalf("trigger.SetEnabled unexpected error: %s", err)
	}
	if trig.Enabled() {
		t.Error("expected trigger.Enabled() to be false after trigger.SetEnabled(false)")
	}
	triggerBytes, err := json.Marshal(trig)
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
	if triggerType != trig.Type().String() {
		t.Fatalf("json.Marshal error, expected marshalled type %q to match trigger.Type() %q", triggerType, trig.Type())
	}
	triggerObj["type"] = "assert test trigger type"
	triggerBytes, err = json.Marshal(triggerObj)
	if err != nil {
		t.Fatalf("json.Unmarshal unexpected error: %s", err)
	}
	if err := json.Unmarshal(triggerBytes, trig); err != nil {
		if err != nil {
			t.Fatalf("json.Unmarshal unexpected error: %s", err)
		}
	}
	if triggerObj["type"] != trig.Type().String() {
		t.Fatalf("json.Unmarshal error, expected unmarshaled type %s to match %s", trig.Type(), triggerObj["type"])
	}
}
