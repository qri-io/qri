package hook_test

import (
	"testing"

	"github.com/qri-io/qri/automation/hook"
	"github.com/qri-io/qri/automation/spec"
)

func TestRuntimeHook(t *testing.T) {
	rh := hook.NewRuntimeHook("testing payload")
	spec.AssertHook(t, rh)
}
