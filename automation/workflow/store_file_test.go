package workflow

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qri-io/qri/event"
)

func TestFileStore(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "TestStoreFile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	newStore := func() Store {
		store, err := NewFileStore(filepath.Join(tmp, "workflows.json"), event.NilBus)
		if err != nil {
			t.Fatalf("creating new store: %s", err)
		}
		return store
	}
	RunWorkflowStoreTests(t, newStore)
}

func TestSubscribe(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "TestStoreFile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	ctx := context.Background()
	bus := event.NewBus(ctx)
	store, err := NewFileStore(filepath.Join(tmp, "workflows.json"), bus)
	if err != nil {
		t.Fatalf("creating new store: %s", err)
	}

	w, err := NewCronWorkflow("w", "owner_ID", "dataset_ID", "R/PT1H")
	if err != nil {
		t.Fatalf("creating workflow: %s", err)
	}
	wID := w.ID
	if err := store.PutWorkflow(ctx, w); err != nil {
		t.Fatalf("putting workflow into store: %s", err)
	}
	expect := w.Copy()
	expect.Status = "running"

	bus.Publish(ctx, ETWorkflowStarted, expect)
	// Better way to ensure we have given enough time for the store get the event & update?
	<-time.After(time.Second)

	got, err := store.GetWorkflow(ctx, wID)
	if err != nil {
		t.Fatalf("getting workflow from store: %s", err)
	}

	if diff := CompareWorkflows(expect, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}

	expect.Status = "failed"

	bus.Publish(ctx, ETWorkflowCompleted, expect)
	// Better way to ensure we have given enough time for the store get the event & update?
	<-time.After(time.Second)

	got, err = store.GetWorkflow(ctx, wID)
	if err != nil {
		t.Fatalf("getting workflow from store: %s", err)
	}
	if diff := CompareWorkflows(expect, got); diff != "" {
		t.Errorf("workflow mismatch (-want +got):\n%s", diff)
	}
}

// func TestFbWorkflowStore(t *testing.T) {
// 	tmp, err := ioutil.TempDir(os.TempDir(), "TestFsWorkflowStore")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer os.RemoveAll(tmp)
// 	newStore := func() Store {
// 		return NewFlatbufferWorkflowStore(filepath.Join(tmp, "workflows.dat"))
// 	}
// 	RunWorkflowStoreTests(t, newStore)
// }

// func BenchmarkFbWorkflowStore(b *testing.B) {
// 	ctx := context.Background()
// 	js := make(workflows, 1000)
// 	for i := range js {
// 		js[i] = &Workflow{
// 			Name:        fmt.Sprintf("workflow_%d", i),
// 			Type:        JTDataset,
// 			Periodicity: mustRepeatingInterval("R/P1H"),
// 		}
// 	}

// 	tmp, err := ioutil.TempDir(os.TempDir(), "TestFsWorkflowStore")
// 	if err != nil {
// 		b.Fatal(err)
// 	}

// 	defer os.RemoveAll(tmp)
// 	store := NewFlatbufferWorkflowStore(filepath.Join(tmp, "workflows.dat"))

// 	b.ResetTimer()

// 	for i := 0; i < b.N; i++ {
// 		if err := store.PutWorkflows(ctx, js...); err != nil {
// 			b.Fatal(err)
// 		}
// 		if _, err := store.ListWorkflows(ctx, 0, -1); err != nil {
// 			b.Fatal(err)
// 		}
// 	}
// }
