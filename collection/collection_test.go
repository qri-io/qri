package collection_test

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/collection/spec"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	profiletest "github.com/qri-io/qri/profile/test"
)

var constructor = func(ctx context.Context, bus event.Bus) (collection.Set, error) {
	return collection.NewLocalSet(ctx, bus, "")
}

func TestLocalCollection(t *testing.T) {
	spec.AssertWritableCollectionSpec(t, constructor)
}

func TestLocalCollectionEvents(t *testing.T) {
	spec.AssertCollectionEventListenerSpec(t, constructor)
}

func TestCollectionPersistence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dir, err := ioutil.TempDir("", "qri_test_local_collection_saving")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	c, err := collection.NewLocalSet(ctx, event.NilBus, dir)
	if err != nil {
		t.Fatal(err)
	}

	wc := c.(collection.WritableSet)

	kermit := profiletest.GetProfile("kermit")
	missPiggy := profiletest.GetProfile("miss_piggy")

	item1 := dsref.VersionInfo{
		ProfileID:  kermit.ID.Encode(),
		InitID:     "muppet_names_init_id",
		Username:   "kermit",
		Name:       "muppet_names",
		CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err = wc.Put(ctx, kermit.ID, item1); err != nil {
		t.Error(err)
	}

	item2 := dsref.VersionInfo{
		ProfileID:  missPiggy.ID.Encode(),
		InitID:     "secret_muppet_friends_init_id",
		Username:   "miss_piggy",
		Name:       "secret_muppet_friends",
		CommitTime: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
	}
	if err = wc.Put(ctx, missPiggy.ID, item2); err != nil {
		t.Error(err)
	}

	// create a new collection to rely on persistence
	c, err = collection.NewLocalSet(ctx, event.NilBus, dir)
	if err != nil {
		t.Error(err)
	}

	got, err := c.List(ctx, kermit.ID, params.ListAll)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff([]dsref.VersionInfo{item1}, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}

	got, err = c.List(ctx, missPiggy.ID, params.ListAll)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff([]dsref.VersionInfo{item2}, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}

const myPid = "QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"

func TestInvalidIDFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dir, err := ioutil.TempDir("", "qri_test_invalid_id_fails")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	c, err := collection.NewLocalSet(ctx, event.NilBus, dir)
	if err != nil {
		t.Fatal(err)
	}
	wc := c.(collection.WritableSet)

	// Construct an invalid profileID (it's base58 encoded), which
	// should result in an error
	info := dsref.VersionInfo{
		ProfileID:  myPid,
		InitID:     "some_init_id",
		Username:   "user",
		Name:       "my_ds",
		CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	err = wc.Put(ctx, myPid, info)
	if err == nil {
		t.Fatal("expected to get an error, did not get one")
	}
	expectErr := `profile.ID invalid, was double encoded as "9tmzz8FC9hjBrY1J9NFFt4gjAzGZWCGrKwB4pcdwuSHC7Y4Y7oPPAkrV48ryPYu". do not pass a base64 encoded string, instead use IDB58Decode(b64encodedID)`
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect: %s, got: %s", expectErr, err)
	}
}

