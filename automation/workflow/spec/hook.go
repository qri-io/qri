package spec

import (
	"encoding/json"
	"testing"

	"github.com/qri-io/qri/automation/workflow"
)

func AssertHook(t *testing.T, hook workflow.Hook) {
	eventType, _ := hook.Event()
	if eventType == "" {
		t.Errorf("Event method must return a non-empty event.Type")
	}
	if hook.Type() == "" {
		t.Error("Type method must return a non-empty HookType")
	}
	if err := hook.SetEnabled(true); err != nil {
		t.Fatalf("hook.SetEnabled unexpected error: %s", err)
	}
	if !hook.Enabled() {
		t.Error("expected hook.Enabled() to be true after hook.SetEnabled(true)")
	}
	if err := hook.SetEnabled(false); err != nil {
		t.Fatalf("hook.SetEnabled unexpected error: %s", err)
	}
	if hook.Enabled() {
		t.Error("expected hook.Enabled() to be false after hook.SetEnabled(false)")
	}
	hookBytes, err := json.Marshal(hook)
	if err != nil {
		t.Fatalf("json.Marshal unexpected error: %s", err)
	}
	hookObj := map[string]interface{}{}
	if err := json.Unmarshal(hookBytes, &hookObj); err != nil {
		t.Fatalf("json.Unmarshal unexpected error: %s", err)
	}
	hookType, ok := hookObj["type"]
	if !ok {
		t.Fatal("json.Marshal error, expected Type field to exist")
	}
	if hookType != hook.Type().String() {
		t.Fatalf("json.Marshal error, expected marshalled type %q to match hook.Type() %q", hookType, hook.Type())
	}
	hookObj["type"] = "assert test hook type"
	hookBytes, err = json.Marshal(hookObj)
	if err != nil {
		t.Fatalf("json.Unmarshal unexpected error: %s", err)
	}
	if err := json.Unmarshal(hookBytes, hook); err != nil {
		if err != nil {
			t.Fatalf("json.Unmarshal unexpected error: %s", err)
		}
	}
	if hookObj["type"] != hook.Type().String() {
		t.Fatalf("json.Unmarshal error, expected unmarshaled type %s to match %s", hook.Type(), hookObj["type"])
	}
}
