package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/qri-io/qri/profile"
)

func TestWorkflowValidate(t *testing.T) {
	now := time.Now()
	ownerID := profile.ID("profile_id")
	cases := []struct {
		description string
		workflow    *Workflow
		expected    error
	}{
		{"nil workflow", nil, ErrNilWorkflow},
		{"no id", &Workflow{}, ErrNoWorkflowID},
		{"no dataset id", &Workflow{ID: "test_id"}, ErrNoDatasetID},
		{"no owner id", &Workflow{ID: "test_id", DatasetID: "dataset_id"}, ErrNoOwnerID},
		{"no created time", &Workflow{ID: "test_id", DatasetID: "dataset_id", OwnerID: ownerID}, ErrNilCreated},
		{"no error", &Workflow{ID: "test_id", DatasetID: "dataset_id", OwnerID: ownerID, Created: &now}, nil},
	}
	for _, c := range cases {
		got := c.workflow.Validate()
		if !errors.Is(c.expected, got) {
			t.Errorf("validate workflow case %q: expected %q, got %q", c.description, c.expected, got)
		}
	}
}
