package spec

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/qri-io/qri/automation/hook"
)

// AssertHook confirms the expected behavior of a hook.Hook Interface
// implementation
func AssertHook(t *testing.T, h hook.Hook) {
	eventType, _ := h.Event()
	if eventType == "" {
		t.Errorf("Event method must return a non-empty event.Type")
	}
	if h.Type() == "" {
		t.Error("Type method must return a non-empty hook.Type")
	}
	if err := h.SetEnabled(true); err != nil {
		t.Fatalf("hook.SetEnabled unexpected error: %s", err)
	}
	if !h.Enabled() {
		t.Error("expected hook.Enabled() to be true after hook.SetEnabled(true)")
	}
	if err := h.SetEnabled(false); err != nil {
		t.Fatalf("hook.SetEnabled unexpected error: %s", err)
	}
	if h.Enabled() {
		t.Error("expected hook.Enabled() to be false after hook.SetEnabled(false)")
	}
	hBytes, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("json.Marshal unexpected error: %s", err)
	}
	hObj := map[string]interface{}{}
	if err := json.Unmarshal(hBytes, &hObj); err != nil {
		t.Fatalf("json.Unmarshal unexpected error: %s", err)
	}
	hType, ok := hObj["type"]
	if !ok {
		t.Fatal("json.Marshal error, expected Type field to exist")
	}
	if hType != h.Type() {
		t.Fatalf("json.Marshal error, expected marshalled type %q to match hook.Type() %q", hType, h.Type())
	}

	hObj["type"] = "bad hook type"
	hBytes, err = json.Marshal(hObj)
	if err != nil {
		t.Fatalf("json.Marshal unexpected error: %s", err)
	}
	if err := json.Unmarshal(hBytes, h); !errors.Is(err, hook.ErrUnexpectedType) {
		t.Fatalf("json.Unmarshal should emit a `hook.ErrUnexpectedType` error if the given type does not match the hook.Type of the Hook")
	}
}
