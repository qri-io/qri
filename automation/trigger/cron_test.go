package trigger_test

import (
	"testing"

	"github.com/qri-io/qri/automation/spec"
	"github.com/qri-io/qri/automation/trigger"
)

func TestCronTrigger(t *testing.T) {
	opts := map[string]interface{}{
		"type":        trigger.CronTriggerType,
		"id":          "test_1",
		"active":      true,
		"periodicity": "R/2021-07-13T21:15:00.000Z/P1H",
	}
	ct, err := trigger.NewCronTrigger(opts)
	if err != nil {
		t.Fatal(err)
	}
	spec.AssertTrigger(t, ct)
}
