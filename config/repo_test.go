package config

import (
	"testing"
)

func TestRepoValidate(t *testing.T) {
	err := DefaultRepo().Validate()
	if err != nil {
		t.Errorf("error validating default repo: %s", err)
	}
}
