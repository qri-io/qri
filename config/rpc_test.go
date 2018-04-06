package config

import (
	"testing"
)

func TestRPCValidate(t *testing.T) {
	err := DefaultRPC().Validate()
	if err != nil {
		t.Errorf("error validating default rpc: %s", err)
	}
}
