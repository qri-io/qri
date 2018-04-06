package config

import (
	"testing"
)

func TestLoggingValidate(t *testing.T) {
	err := DefaultLogging().Validate()
	if err != nil {
		t.Errorf("error validating default logging: %s", err)
	}
}
