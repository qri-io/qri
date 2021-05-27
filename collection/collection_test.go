package collection_test

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/collection/spec"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/params"
	profiletest "github.com/qri-io/qri/profile/test"
)

func TestLocalCollection(t *testing.T) {
	spec.AssertWritableCollectionSpec(t, func(ctx context.Context, bus event.Bus) (collection.Collection, error) {
		return collection.NewLocalCollection(ctx, bus, "")
	})
}

func TestCollectionPersistence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dir, err := ioutil.TempDir("", "qri_test_local_collection_saving")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	c, err := collection.NewLocalCollection(ctx, event.NilBus, dir)
	if err != nil {
		t.Fatal(err)
	}

	wc := c.(collection.Writable)

	kermit := profiletest.GetProfile("kermit")
	missPiggy := profiletest.GetProfile("miss_piggy")

	item1 := collection.Item{
		ProfileID:  kermit.ID,
		InitID:     "muppet_names_init_id",
		Username:   "kermit",
		Name:       "muppet_names",
		CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err = wc.Put(ctx, kermit.ID, item1); err != nil {
		t.Error(err)
	}

	item2 := collection.Item{
		ProfileID:  missPiggy.ID,
		InitID:     "secret_muppet_friends_init_id",
		Username:   "miss_piggy",
		Name:       "secret_muppet_friends",
		CommitTime: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
	}
	if err = wc.Put(ctx, missPiggy.ID, item2); err != nil {
		t.Error(err)
	}

	// create a new collection to rely on persistence
	c, err = collection.NewLocalCollection(ctx, event.NilBus, dir)
	if err != nil {
		t.Error(err)
	}

	got, err := c.List(ctx, kermit.ID, params.ListAll)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff([]collection.Item{item1}, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}

	got, err = c.List(ctx, missPiggy.ID, params.ListAll)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff([]collection.Item{item2}, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
