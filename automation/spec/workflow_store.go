package spec

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/automation/hook"
	"github.com/qri-io/qri/automation/trigger"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/profile"
)

// AssertWorkflowStore confirms the expected behavior of a workflow.Store Interface
// implementation
func AssertWorkflowStore(t *testing.T, store workflow.Store) {
	ctx := context.Background()
	seedStr := "workflow assert store seed string used for testing in the workflow package"
	workflow.SetIDRand(strings.NewReader(seedStr))
	now := time.Now()

	aliceDatasetID := "alice_dataset_id"
	aliceProID := profile.ID("alice_pro_id")
	aliceTrigger := map[string]interface{}{"type": trigger.RuntimeType}
	aliceHook := map[string]interface{}{"type": hook.RuntimeType}
	alice := &workflow.Workflow{
		DatasetID: aliceDatasetID,
		OwnerID:   aliceProID,
		Created:   &now,
		Triggers:  []map[string]interface{}{aliceTrigger},
		Hooks:     []map[string]interface{}{aliceHook},
	}
	got, err := store.Put(ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == "" {
		t.Errorf("store.Put error: a workflow with no ID is considered a new workflow. The workflow.Store should create an ID and return the workflow with the generated ID")
	}
	alice.ID = got.ID
	aliceID := alice.ID
	if diff := cmp.Diff(alice, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	got, err = store.Get(ctx, aliceID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(alice, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	got, err = store.GetByDatasetID(ctx, alice.DatasetID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(alice, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	// store.Put error checking
	if _, err := store.Put(ctx, &workflow.Workflow{DatasetID: aliceDatasetID}); !errors.Is(err, workflow.ErrWorkflowForDatasetExists) {
		t.Errorf("Put method must emit `workflow.ErrWorkflowForDatasetExists` error if a workflow for the given DatasetID already exists")
	}

	brittDatasetID := "britt_dataset_id"
	brittProID := profile.ID("britt_pro_id")
	brittTrigger := map[string]interface{}{"type": trigger.RuntimeType}
	brittHook := map[string]interface{}{"type": hook.RuntimeType}
	britt := &workflow.Workflow{
		DatasetID: brittDatasetID,
		OwnerID:   brittProID,
		Created:   &now,
		Triggers:  []map[string]interface{}{brittTrigger},
		Hooks:     []map[string]interface{}{brittHook},
	}
	got, err = store.Put(ctx, britt)
	if err != nil {
		t.Fatal(err)
	}

	britt.ID = got.ID
	if diff := cmp.Diff(britt, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	wfs, err := store.List(ctx, -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(wfs) != 2 {
		t.Fatalf("store.List count mismatch, expected 2 workflows, got %d", len(wfs))
	}

	deployed, err := store.ListDeployed(ctx, -1, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(deployed) != 0 {
		t.Fatalf("store.ListDeployed count mismatch, expected 0 workflows, got %d", len(deployed))
	}

	aliceUpdated := &workflow.Workflow{
		ID:        aliceID,
		DatasetID: alice.DatasetID,
		OwnerID:   alice.OwnerID,
		Active:    true,
		Created:   &now,
		Triggers:  []map[string]interface{}{aliceTrigger, brittTrigger},
		Hooks:     []map[string]interface{}{aliceHook, brittHook},
	}
	_, err = store.Put(ctx, aliceUpdated)
	if err != nil {
		t.Fatal(err)
	}

	got, err = store.Get(ctx, aliceID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(aliceUpdated, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	deployed, err = store.ListDeployed(ctx, -1, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(deployed) != 1 {
		t.Fatalf("store.ListDeployed count mismatch, expected 1 workflow, got %d", len(deployed))
	}
	if diff := cmp.Diff(aliceUpdated, deployed[0]); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	err = store.Remove(ctx, aliceID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Get(ctx, aliceID)
	if !errors.Is(err, workflow.ErrNotFound) {
		t.Errorf("store.Get error mistmatch, expected %q, got %q", workflow.ErrNotFound, err)
	}
}

// AssertWorkflowLister confirms the expected behavior of a workflow.Lister Interface
// implementation
func AssertWorkflowLister(t *testing.T, store workflow.Store) {
	// set up
	workflow.SetIDRand(strings.NewReader(strings.Repeat("Lorem ipsum dolor sit amet", 20)))
	ctx := context.Background()
	expectedAllWorkflows := [10]*workflow.Workflow{}
	expectedDeployedWorkflows := [5]*workflow.Workflow{}

	proID := profile.ID("profile_id")
	for i := 0; i < 10; i++ {
		now := time.Now()
		wf, err := store.Put(ctx, &workflow.Workflow{
			DatasetID: fmt.Sprintf("dataset_%d", i),
			OwnerID:   proID,
			Created:   &now,
		})
		if err != nil {
			t.Fatal(err)
		}
		if i%2 == 0 {
			wf.Active = true
			expectedDeployedWorkflows[4-(i/2)] = wf
		}
		expectedAllWorkflows[9-i] = wf
	}

	// error cases
	errCases := []errTestCase{
		{"negative limit", -10, 0, "limit of -10 is out of bounds"},
		{"negative offset", 0, -1, "offset of -1 is out of bounds"},
	}

	runListErrTestCases(ctx, t, "List", store.List, errCases)
	runListErrTestCases(ctx, t, "ListDeployed", store.ListDeployed, errCases)

	// empty list cases
	emptyCases := []emptyTestCase{
		{"offset out of bounds", 10, 100},
		{"zero limit", 0, 0},
	}

	runListEmptyTestCases(ctx, t, "List", store.List, emptyCases)
	runListEmptyTestCases(ctx, t, "ListDeployed", store.ListDeployed, emptyCases)

	// working cases
	cases := []expectedTestCase{
		{"get all", -1, 0, expectedAllWorkflows[:]},
		{"get first 4", 4, 0, expectedAllWorkflows[0:4]},
		{"get next 4", 4, 4, expectedAllWorkflows[4:8]},
		{"get last 2", 4, 8, expectedAllWorkflows[8:]},
	}

	runListExpectedTestCases(ctx, t, "List", store.List, cases)

	cases = []expectedTestCase{
		{"get all", -1, 0, expectedDeployedWorkflows[:]},
		{"get first 2", 2, 0, expectedDeployedWorkflows[0:2]},
		{"get next 2", 2, 2, expectedDeployedWorkflows[2:4]},
		{"get last 1", 2, 4, expectedDeployedWorkflows[4:]},
	}
	runListExpectedTestCases(ctx, t, "ListDeployed", store.ListDeployed, cases)
}

type expectedTestCase struct {
	description string
	limit       int
	offset      int
	expected    []*workflow.Workflow
}

func runListExpectedTestCases(ctx context.Context, t *testing.T, fnName string, fn listFunc, cases []expectedTestCase) {
	for _, c := range cases {
		got, err := fn(ctx, c.limit, c.offset)
		if err != nil {
			t.Errorf("%s case %s: unexpected error %w", fnName, c.description, err)
			continue
		}
		if diff := cmp.Diff(c.expected, got); diff != "" {
			t.Errorf("%s case %s: workflow list mismatch (-want +got):\n%s", fnName, c.description, diff)
		}
	}
}

type listFunc func(ctx context.Context, limit, offset int) ([]*workflow.Workflow, error)

type errTestCase struct {
	description string
	limit       int
	offset      int
	errMsg      string
}

func runListErrTestCases(ctx context.Context, t *testing.T, fnName string, fn listFunc, cases []errTestCase) {
	for _, c := range cases {
		_, err := fn(ctx, c.limit, c.offset)
		if err == nil {
			t.Errorf("%s case %s: error mismatch, expected %q, got no error", fnName, c.description, c.errMsg)
			continue
		}
		if err.Error() != c.errMsg {
			t.Errorf("%s case %s: error mismatch, expected %q, got %q", fnName, c.description, c.errMsg, err)
		}
	}
}

type emptyTestCase struct {
	description string
	limit       int
	offset      int
}

func runListEmptyTestCases(ctx context.Context, t *testing.T, fnName string, fn listFunc, cases []emptyTestCase) {
	expected := []*workflow.Workflow{}
	for _, c := range cases {
		got, err := fn(ctx, c.limit, c.offset)
		if err != nil {
			t.Errorf("%s case %s: unexpected error %q", fnName, c.description, err)
			continue
		}
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Errorf("%s case %s: workflow list mismatch (-want +got):\n%s", fnName, c.description, diff)
		}
	}
}
