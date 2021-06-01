package trigger_test

import (
	"testing"

	"github.com/qri-io/qri/automation/spec"
	"github.com/qri-io/qri/automation/trigger"
)

func TestRuntimeTrigger(t *testing.T) {
	rt := trigger.NewRuntimeTrigger()
	spec.AssertTrigger(t, rt)
}
