package workflow

import (
	"context"
	"errors"
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
}

func AssertStore(t *testing.T, store Store) {
	aliceDatasetID := "alice_dataset_id"
	aliceProID := profile.ID("alice_pro_id")
	aliceSeedStr := "alice workflow assert store seed string"
	SetIDRand(strings.NewReader(aliceSeedStr))
	aliceID := NewID()
	// aliceMemTrigger := NewMemTrigger()
	// aliceMemHook := NewMemHook()
	alice := &Workflow{
		ID:        aliceID,
		DatasetID: aliceDatasetID,
		OwnerID:   aliceProID,
		// Triggers:  []Trigger{aliceMemTrigger},
		// Hooks:     []Hook{aliceMemHook},
	}
	// got, err := store.Create(aliceDatasetID, aliceProID, []Trigger{aliceMemTrigger}, []Hook{aliceMemHook})
	SetIDRand(strings.NewReader(aliceSeedStr))
	got, err := store.Create(aliceDatasetID, aliceProID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(alice, got, cmpopts.IgnoreFields(Workflow{}, "Created")); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	got, err = store.Get(aliceID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(alice, got, cmpopts.IgnoreFields(Workflow{}, "Created")); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	got, err = store.GetDatasetWorkflow(alice.DatasetID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(alice, got, cmpopts.IgnoreFields(Workflow{}, "Created")); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	// _, err = store.Create(aliceDatasetID, "new_profile_ID", []Trigger{}, []Hook{})
	_, err = store.Create(aliceDatasetID, "new_profile_ID")
	if !errors.Is(err, ErrWorkflowForDatasetExists) {
		t.Errorf("store.Create error mismatch, expected %q, got %q", ErrWorkflowForDatasetExists, err)
	}

	brittDatasetID := "britt_dataset_id"
	brittProID := profile.ID("britt_pro_id")
	brittSeedStr := "britt workflow assert store seed string"
	SetIDRand(strings.NewReader(brittSeedStr))
	brittID := NewID()
	// brittMemTrigger .:= NewMemTrigger()
	// britMemHook := NewMemHook()
	britt := &Workflow{
		ID:        brittID,
		DatasetID: brittDatasetID,
		OwnerID:   brittProID,
		// Triggers:  []Trigger{brittMemTrigger},
		// Hooks:     []Hook{brittMemTrigger},
	}
	// got, err = store.Create(brittDatasetID, brittProID, []Trigger{brittMemTrigger}, []Hook{brittMemHook})
	SetIDRand(strings.NewReader(brittSeedStr))
	got, err = store.Create(brittDatasetID, brittProID)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(britt, got, cmpopts.IgnoreFields(Workflow{}, "Created")); diff != "" {
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
		// Triggers:  []Trigger{aliceMemTrigger, brittMemTrigger},
		// Hooks:     []Hook{aliceMemHook, brittMemHook},
	}
	err = store.Update(aliceUpdated)
	if err != nil {
		t.Fatal(err)
	}

	got, err = store.Get(aliceID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(aliceUpdated, got, cmpopts.IgnoreFields(Workflow{}, "Created")); diff != "" {
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
	if diff := cmp.Diff(britt, deployed[0], cmpopts.IgnoreFields(Workflow{}, "Created")); diff != "" {
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
	// TODO (ramfox): replace with errors.Is?
	if err != ErrNotFound {
		t.Errorf("store.Get error mistmatch, expected %q, got %q", ErrNotFound, err)
	}
}
