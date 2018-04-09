package config

import (
	"testing"
)

func TestCLIValidate(t *testing.T) {
	err := DefaultCLI().Validate()
	if err != nil {
		t.Errorf("error validating default cli: %s", err)
	}
}
