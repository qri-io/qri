package run_test

import (
	"testing"

	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/automation/spec"
)

func TestMemStore(t *testing.T) {
	store := run.NewMemStore()
	spec.AssertRunStore(t, store)
}
