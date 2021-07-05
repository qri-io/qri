package config

import (
	"testing"
)

func TestAutomationValidate(t *testing.T) {
	err := DefaultAutomation().Validate()
	if err != nil {
		t.Errorf("error validating default api: %s", err)
	}
}

func TestAutomationCopy(t *testing.T) {
	a := DefaultAutomation()
	b := a.Copy()

	a.Enabled = !a.Enabled
	a.RunStoreMaxSize = "foo"

	if a.Enabled == b.Enabled {
		t.Errorf("Enabled fields should not match")
	}
	if a.RunStoreMaxSize == b.RunStoreMaxSize {
		t.Errorf("RunStoreMaxSize fields should not match")
	}
}
