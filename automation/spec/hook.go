package spec

import (
	"encoding/json"
	"testing"

	"github.com/qri-io/qri/automation/hook"
)

func AssertHook(t *testing.T, h hook.Hook) {
	eventType, _ := h.Event()
	if eventType == "" {
		t.Errorf("Event method must return a non-empty event.Type")
	}
	if h.Type() == "" {
		t.Error("Type method must return a non-empty HookType")
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
	if hType != h.Type().String() {
		t.Fatalf("json.Marshal error, expected marshalled type %q to match hook.Type() %q", hType, h.Type())
	}
	hObj["type"] = "assert test hook type"
	hBytes, err = json.Marshal(hObj)
	if err != nil {
		t.Fatalf("json.Unmarshal unexpected error: %s", err)
	}
	if err := json.Unmarshal(hBytes, h); err != nil {
		if err != nil {
			t.Fatalf("json.Unmarshal unexpected error: %s", err)
		}
	}
	if hObj["type"] != h.Type().String() {
		t.Fatalf("json.Unmarshal error, expected unmarshaled type %s to match %s", h.Type(), hObj["type"])
	}
}
