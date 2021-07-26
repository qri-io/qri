package trigger_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/automation/trigger"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/profile"
)

func TestSet(t *testing.T) {
	s := trigger.NewSet(trigger.RuntimeType, trigger.NewRuntimeTrigger)
	if err := s.Add(&workflow.Workflow{OwnerID: profile.ID("ownerID")}); !errors.Is(err, trigger.ErrEmptyWorkflowID) {
		t.Errorf("Add - source with no WorkflowID - expected error %q, got %q", trigger.ErrEmptyWorkflowID, err)
	}
	if err := s.Add(&workflow.Workflow{ID: "workflowID"}); !errors.Is(err, trigger.ErrEmptyOwnerID) {
		t.Errorf("Add - source with no OwnerID - expected error %q, got %q", trigger.ErrEmptyOwnerID, err)
	}
	if err := s.Add(&workflow.Workflow{OwnerID: profile.ID("ownerID"), ID: "workflowID"}); err != nil {
		t.Errorf("Add - unexpected error: %s", err)
	}
	if diff := cmp.Diff(map[profile.ID]map[string][]trigger.Trigger{}, s.Active()); diff != "" {
		t.Errorf("active triggers mismatch (-want +got):\n%s", diff)
	}
	alice := profile.ID("alice")
	sourceA := &workflow.Workflow{
		Active:  true,
		OwnerID: alice,
		ID:      "workflow1",
		Triggers: []map[string]interface{}{
			{
				"id":     "trigger1",
				"active": true,
				"type":   trigger.RuntimeType,
			},
			{
				"id":     "trigger_not_active",
				"active": false,
				"type":   trigger.RuntimeType,
			},
		},
	}
	sourceB := &workflow.Workflow{
		Active:  true,
		OwnerID: alice,
		ID:      "workflow2",
	}
	sourceC := &workflow.Workflow{
		Active:  true,
		OwnerID: alice,
		ID:      "workflow3",
		Triggers: []map[string]interface{}{
			{
				"id":     "trigger2",
				"active": true,
				"type":   trigger.RuntimeType,
			},
			{
				"id":     "trigger3",
				"active": true,
				"type":   trigger.RuntimeType,
			},
		},
	}
	bob := profile.ID("bob")
	sourceD := &workflow.Workflow{
		Active:  true,
		OwnerID: bob,
		ID:      "workflow4",
		Triggers: []map[string]interface{}{
			{
				"id":     "trigger4",
				"active": true,
				"type":   trigger.RuntimeType,
			},
		},
	}
	sourceE := &workflow.Workflow{
		Active:  false,
		OwnerID: bob,
		ID:      "workflow5",
		Triggers: []map[string]interface{}{
			{
				"id":     "trigger_not_active_workflow",
				"active": true,
				"type":   trigger.RuntimeType,
			},
		},
	}

	prevNewID := trigger.NewID
	defer func() {
		trigger.NewID = prevNewID
	}()
	triggerIDs := []string{
		"trigger1",
		"trigger2",
		"trigger3",
		"trigger4",
	}
	triggerIDIndex := 0
	trigger.NewID = func() string {
		if triggerIDIndex >= len(triggerIDs) {
			t.Fatal("more triggers created than ids allocated")
		}
		id := triggerIDs[triggerIDIndex]
		triggerIDIndex++
		return id
	}
	triggers := []trigger.Trigger{}
	for i := 0; i < len(triggerIDs); i++ {
		trig := trigger.NewEmptyRuntimeTrigger()
		trig.SetActive(true)
		triggers = append(triggers, trig)
	}

	expected := map[profile.ID]map[string][]trigger.Trigger{
		alice: {
			"workflow1": triggers[:1],
			"workflow3": triggers[1:3],
		},
		bob: {
			"workflow4": triggers[3:],
		},
	}
	if err := s.Add(sourceA, sourceB, sourceC, sourceD, sourceE); err != nil {
		t.Fatalf("Add unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, s.Active(), cmp.AllowUnexported(trigger.RuntimeTrigger{})); diff != "" {
		t.Errorf("active triggers mismatch (-want +got):\n%s", diff)
	}

	if exists := s.Exists(sourceD); !exists {
		t.Fatal("Exists error, expected source D to exist")
	}

	// removing the triggers for an ownerID that has no more workflows
	// should remove the entry in the store
	sourceD.Triggers = []map[string]interface{}{}
	delete(expected, bob)
	if err := s.Add(sourceD); err != nil {
		t.Fatalf("Add unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, s.Active(), cmp.AllowUnexported(trigger.RuntimeTrigger{})); diff != "" {
		t.Errorf("active triggers mismatch (-want +got):\n%s", diff)
	}

	if exists := s.Exists(sourceD); exists {
		t.Fatal("Exists error, expected source D to NOT exists")
	}

	sourceA.Triggers = []map[string]interface{}{
		sourceA.Triggers[0],
		sourceC.Triggers[0],
		sourceC.Triggers[1],
	}
	expected[alice]["workflow1"] = triggers[:3]
	if err := s.Add(sourceA); err != nil {
		t.Fatalf("Add unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, s.Active(), cmp.AllowUnexported(trigger.RuntimeTrigger{})); diff != "" {
		t.Errorf("active triggers mismatch (-want +got):\n%s", diff)
	}

	sourceA.Triggers = sourceA.Triggers[:1]
	if exists := s.Exists(sourceA); exists {
		t.Fatal("Exists error, expected source A (with incorrect triggers) to NOT exist")
	}

	sourceC.Triggers = []map[string]interface{}{}
	delete(expected[alice], "workflow3")
	if err := s.Add(sourceC); err != nil {
		t.Fatalf("Add unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, s.Active(), cmp.AllowUnexported(trigger.RuntimeTrigger{})); diff != "" {
		t.Errorf("active triggers mismatch (-want +got):\n%s", diff)
	}

	sourceA.Triggers = []map[string]interface{}{}
	delete(expected, alice)
	if err := s.Add(sourceA); err != nil {
		t.Fatalf("Add unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, s.Active(), cmp.AllowUnexported(trigger.RuntimeTrigger{})); diff != "" {
		t.Errorf("active triggers mismatch (-want +got):\n%s", diff)
	}
}
