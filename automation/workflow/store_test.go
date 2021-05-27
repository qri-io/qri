package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/qri/profile"
)

// func TestWorkflowAdvance(t *testing.T)
// func TestWorkflowComplete(t *testing.T)

func TestMemStoreIntegration(t *testing.T) {
	store := NewMemStore()
	AssertStore(t, store)
	store = NewMemStore()
	AssertLister(t, store)
}

func AssertStore(t *testing.T, store Store) {
	aliceDatasetID := "alice_dataset_id"
	aliceProID := profile.ID("alice_pro_id")
	aliceSeedStr := "alice workflow assert store seed string used for testing in the workflow package"
	SetIDRand(strings.NewReader(aliceSeedStr))
	aliceID := NewID()
	aliceTestTrigger := NewTestTrigger(aliceID)
	aliceTestHook := NewTestHook("hook payload")
	alice := &Workflow{
		ID:        aliceID,
		DatasetID: aliceDatasetID,
		OwnerID:   aliceProID,
		Triggers:  []Trigger{aliceTestTrigger},
		Hooks:     []Hook{aliceTestHook},
	}
	SetIDRand(strings.NewReader(aliceSeedStr))
	got, err := store.Create(aliceDatasetID, aliceProID, []Trigger{aliceTestTrigger}, []Hook{aliceTestHook})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(alice, got, cmpopts.IgnoreFields(Workflow{}, "Created"), cmp.AllowUnexported(TestTrigger{}, TestHook{})); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	got, err = store.Get(aliceID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(alice, got, cmpopts.IgnoreFields(Workflow{}, "Created"), cmp.AllowUnexported(TestTrigger{}, TestHook{})); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	got, err = store.GetDatasetWorkflow(alice.DatasetID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(alice, got, cmpopts.IgnoreFields(Workflow{}, "Created"), cmp.AllowUnexported(TestTrigger{}, TestHook{})); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	_, err = store.Create(aliceDatasetID, "new_profile_ID", []Trigger{}, []Hook{})
	if !errors.Is(err, ErrWorkflowForDatasetExists) {
		t.Errorf("store.Create error mismatch, expected %q, got %q", ErrWorkflowForDatasetExists, err)
	}

	brittDatasetID := "britt_dataset_id"
	brittProID := profile.ID("britt_pro_id")
	brittSeedStr := "britt workflow assert store seed string used for testing in the workflow package"
	SetIDRand(strings.NewReader(brittSeedStr))
	brittID := NewID()
	brittTestTrigger := NewTestTrigger(brittID)
	brittTestHook := NewTestHook("hook payload")
	britt := &Workflow{
		ID:        brittID,
		DatasetID: brittDatasetID,
		OwnerID:   brittProID,
		Triggers:  []Trigger{brittTestTrigger},
		Hooks:     []Hook{brittTestHook},
	}
	SetIDRand(strings.NewReader(brittSeedStr))
	got, err = store.Create(brittDatasetID, brittProID, []Trigger{brittTestTrigger}, []Hook{brittTestHook})
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(britt, got, cmpopts.IgnoreFields(Workflow{}, "Created"), cmp.AllowUnexported(TestTrigger{}, TestHook{})); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	ctx := context.Background()
	wfs, err := store.List(ctx, -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(wfs) != 2 {
		t.Fatalf("store.List count mismatch, expected 2 workflows, got %d", len(wfs))
	}

	aliceUpdated := &Workflow{
		ID:        aliceID,
		DatasetID: alice.DatasetID,
		OwnerID:   alice.OwnerID,
		Triggers:  []Trigger{aliceTestTrigger, brittTestTrigger},
		Hooks:     []Hook{aliceTestHook, brittTestHook},
	}
	err = store.Update(aliceUpdated)
	if err != nil {
		t.Fatal(err)
	}

	got, err = store.Get(aliceID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(aliceUpdated, got, cmpopts.IgnoreFields(Workflow{}, "Created"), cmp.AllowUnexported(TestTrigger{}, TestHook{})); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	err = store.Deploy(brittID)
	if err != nil {
		t.Fatal(err)
	}

	deployed, err := store.ListDeployed(ctx, -1, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(deployed) != 1 {
		t.Fatalf("store.ListDeployed count mismatch, expected 1 workflow, got %d", len(deployed))
	}
	britt.Deployed = true
	if diff := cmp.Diff(britt, deployed[0], cmpopts.IgnoreFields(Workflow{}, "Created"), cmp.AllowUnexported(TestTrigger{}, TestHook{})); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	err = store.Undeploy(brittID)
	if err != nil {
		t.Fatal(err)
	}

	deployed, err = store.ListDeployed(ctx, -1, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(deployed) != 0 {
		t.Fatalf("store.ListDeployed count mismatch, expected 0 workflows, got %d", len(deployed))
	}

	err = store.Remove(brittID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Get(brittID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("store.Get error mistmatch, expected %q, got %q", ErrNotFound, err)
	}
}

func AssertLister(t *testing.T, store Store) {
	// set up
	SetIDRand(strings.NewReader("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat."))
	ctx := context.Background()
	expectedAllWorkflows := [10]*Workflow{}
	expectedDeployedWorkflows := [5]*Workflow{}
	proID := profile.ID("profile_id")
	for i := 0; i < 10; i++ {
		wf, err := store.Create(fmt.Sprintf("dataset_%d", i), proID, []Trigger{}, []Hook{})
		if err != nil {
			t.Fatal(err)
		}
		if i%2 == 0 {
			wf.Deployed = true
			expectedDeployedWorkflows[4-(i/2)] = wf
		}
		expectedAllWorkflows[9-i] = wf
	}

	// error cases
	errCases := []errTestCase{
		{"negative limit", -10, 0, "limit of -10 is out of bounds"},
		{"negative offset", 0, -1, "offset of -1 is out of bounds"},
	}

	runListErrTestCases(t, ctx, "List", store.List, errCases)
	runListErrTestCases(t, ctx, "ListDeployed", store.ListDeployed, errCases)

	// empty list cases
	emptyCases := []emptyTestCase{
		{"offset out of bounds", 10, 100},
		{"zero limit", 0, 0},
	}

	runListEmptyTestCases(t, ctx, "List", store.List, emptyCases)
	runListEmptyTestCases(t, ctx, "ListDeployed", store.ListDeployed, emptyCases)

	// working cases
	cases := []expectedTestCase{
		{"get all", -1, 0, expectedAllWorkflows[:]},
		{"get first 4", 4, 0, expectedAllWorkflows[0:4]},
		{"get next 4", 4, 4, expectedAllWorkflows[4:8]},
		{"get last 2", 4, 8, expectedAllWorkflows[8:]},
	}

	runListExpectedTestCases(t, ctx, "List", store.List, cases)

	cases = []expectedTestCase{
		{"get all", -1, 0, expectedDeployedWorkflows[:]},
		{"get first 2", 2, 0, expectedDeployedWorkflows[0:2]},
		{"get next 2", 2, 2, expectedDeployedWorkflows[2:4]},
		{"get last 1", 2, 4, expectedDeployedWorkflows[4:]},
	}
	runListExpectedTestCases(t, ctx, "ListDeployed", store.ListDeployed, cases)
}

type expectedTestCase struct {
	description string
	limit       int
	offset      int
	expected    []*Workflow
}

func runListExpectedTestCases(t *testing.T, ctx context.Context, fnName string, fn ListFunc, cases []expectedTestCase) {
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

type ListFunc func(ctx context.Context, limit, offset int) ([]*Workflow, error)

type errTestCase struct {
	description string
	limit       int
	offset      int
	errMsg      string
}

func runListErrTestCases(t *testing.T, ctx context.Context, fnName string, fn ListFunc, cases []errTestCase) {
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

func runListEmptyTestCases(t *testing.T, ctx context.Context, fnName string, fn ListFunc, cases []emptyTestCase) {
	expected := []*Workflow{}
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
