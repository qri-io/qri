package workflow

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDatasetOptionsJSON(t *testing.T) {
	src := &DatasetOptions{
		Title:     "A_Title",
		Message:   "A_Message",
		Recall:    "A_Recall",
		BodyPath:  "A_BodyPath",
		FilePaths: []string{"a", "b", "c"},

		Publish:             true,
		Strict:              true,
		Force:               true,
		ConvertFormatToPrev: true,
		ShouldRender:        true,

		Config:  map[string]string{"a": "a"},
		Secrets: map[string]string{"b": "b"},
	}

	data, err := json.Marshal(src)
	if err != nil {
		t.Fatal(err)
	}

	got := &DatasetOptions{}
	if err := json.Unmarshal(data, got); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(src, got); diff != "" {
		t.Errorf("result mismatch. (-wnt +got):\n%s", diff)
	}
}

func TestWorkflowCopy(t *testing.T) {
	// now := time.Now()
	a := &Workflow{
		ID:   "id",
		Name: "name",
		Type: Type("FOO"),
		// PrevRunStart: &now,
		// RunNumber:    1234567890,
		// RunStart:     &now,
		// RunStop:      &now,
		// RunError:     "oh noes it broke",
		// LogFilePath:  "such filepath",
		// RepoPath:     "such repo path",
		Options: &DatasetOptions{
			FilePaths: []string{"the", "file", "paths"},
		},
	}

	if diff := CompareWorkflows(a, a.Copy()); diff != "" {
		t.Errorf("copy mismatch (-want +got):\n%s", diff)
	}
}

func TestWorkflowsJSON(t *testing.T) {
	workflows := NewWorkflowSet()
	workflows.Add(&Workflow{
		ID:      "workflow1",
		Name:    "workflow_one",
		Type:    JTDataset,
		Options: &DatasetOptions{Title: "Yus"},
	})
	workflows.Add(&Workflow{
		ID:   "workflow2",
		Name: "workflow_two",
		Type: JTShellScript,
	})

	data, err := json.Marshal(workflows)
	if err != nil {
		t.Fatal(err)
	}

	got := []*Workflow{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	for i, j := range got {
		if diff := CompareWorkflows(workflows.set[i], j); diff != "" {
			t.Errorf("workflow %d mismatch (-want +got):\n%s", i, diff)
		}
	}

}
