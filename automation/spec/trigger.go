package spec

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/qri-io/qri/automation/trigger"
)

// AssertTrigger confirms the expected behavior of a trigger.Trigger Interface
// implementation
func AssertTrigger(t *testing.T, trig trigger.Trigger) {
	if trig.Type() == "" {
		t.Error("Type method must return a non-empty trigger.Type")
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
	if err := json.Unmarshal(triggerBytes, &triggerObj); err != nil {
		t.Fatalf("json.Unmarshal unexpected error: %s", err)
	}

	triggerObj["type"] = "bad trigger type"
	triggerBytes, err = json.Marshal(triggerObj)
	if err != nil {
		t.Fatalf("json.Marshal unexpected error: %s", err)
	}
	if err := json.Unmarshal(triggerBytes, trig); !errors.Is(err, trigger.ErrUnexpectedType) {
		t.Fatalf("json.Unmarshal should emit a `trigger.ErrUnexpectedType` error if the given type does not match the trigger.Type of the Trigger")
	}
}
