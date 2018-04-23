package config

import (
	"testing"
)

func TestProfileValidate(t *testing.T) {
	err := DefaultProfile().Validate()
	if err != nil {
		t.Errorf("error validating default profile: %s", err)
	}
}
