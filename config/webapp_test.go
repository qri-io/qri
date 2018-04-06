package config

import (
	"testing"
)

func TestWebappValidate(t *testing.T) {
	err := DefaultWebapp().Validate()
	if err != nil {
		t.Errorf("error validating default webapp: %s", err)
	}
}
