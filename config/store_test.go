package config

import (
	"testing"
)

func TestStoreValidate(t *testing.T) {
	err := DefaultStore().Validate()
	if err != nil {
		t.Errorf("error validating default store: %s", err)
	}
}
