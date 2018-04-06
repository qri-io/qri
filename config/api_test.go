package config

import (
	"testing"
)

func TestAPIValidate(t *testing.T) {
	err := DefaultAPI().Validate()
	if err != nil {
		t.Errorf("error validating default api: %s", err)
	}
}
