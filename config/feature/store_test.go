package feature

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestLocalStore(t *testing.T) {
	path, err := ioutil.TempDir("", "flags")
	if err != nil {
		t.Fatalf("error creating tmp directory: %s", err.Error())
	}
	t.Logf("store: %s", path)

	ffs, err := NewLocalStore(filepath.Join(path, "feature_flag_test.json"))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	flagList, err := ffs.List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	lenFlagList := len(flagList)
	if lenFlagList != len(DefaultFlags) {
		t.Errorf("expected same length of returned feature flags as default: %d got %d", len(flagList), len(DefaultFlags))
	}

	flag, err := ffs.Get(ctx, "BASE")
	if err != nil {
		t.Fatal(err)
	}
	if flag.ID != DefaultFlags["BASE"].ID || flag.Active != DefaultFlags["BASE"].Active {
		t.Errorf("Get didn't return the expected values")
	}

	ffs.Put(ctx, &Flag{
		ID:     "BASE",
		Active: false,
	})
	flag, err = ffs.Get(ctx, "BASE")
	if err != nil {
		t.Fatal(err)
	}
	if flag.Active {
		t.Errorf("flag remained active after turning off")
	}

	ffs.Put(ctx, &Flag{
		ID:     "NEW_BASE",
		Active: true,
	})
	_, err = ffs.Get(ctx, "NEW_BASE")
	if err != nil {
		t.Fatal(err)
	}
	flagList, err = ffs.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(flagList) != lenFlagList+1 {
		t.Errorf("expected flag list to grow with new flag added")
	}

	flag, err = ffs.Get(ctx, "UNKNOWN_FLAG")
	if err != nil {
		t.Fatal(err)
	}
	if flag.Active {
		t.Errorf("expected unknown flags to always return false")
	}
}
