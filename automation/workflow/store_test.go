package workflow_test

import (
	"testing"

	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/automation/workflow/spec"
)

func TestMemStoreIntegration(t *testing.T) {
	store := workflow.NewMemStore()
	spec.AssertStore(t, store)
	store = workflow.NewMemStore()
	spec.AssertLister(t, store)
}
