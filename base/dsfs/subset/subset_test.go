package subset

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base/dsfs"
)

func addMovies(t *testing.T, s cafs.Filestore) string {
	ctx := context.Background()
	prev := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Time{}.UTC() }
	defer func() {
		dsfs.Timestamp = prev
	}()

	tc, err := dstest.NewTestCaseFromDir("testdata/movies")
	if err != nil {
		t.Fatal(err)
	}

	path, err := dsfs.CreateDataset(ctx, s, tc.Input, nil, dstest.PrivKey, true, false, true)
	if err != nil {
		t.Fatal(err)
	}

	return path
}

func TestLoadPreview(t *testing.T) {
	ctx := context.Background()
	s := cafs.NewMapstore()
	path := addMovies(t, s)

	res, err := LoadPreview(ctx, s, path)
	if err != nil {
		t.Error(err)
	}

	expect := "ca0be54642b2b7d0a7c28c0628c8200fe7889f50"
	sum := dstest.DatasetChecksum(res)
	if expect != sum {
		t.Errorf("dataset checksum mismatch. expected: %s, got: %s", expect, sum)
	}
}

func TestPreview(t *testing.T) {
	p := Preview(&dataset.Dataset{})

	expect := "085e607818aae2920e0e4b57c321c3b58e17b85d"
	sum := dstest.DatasetChecksum(p)
	if expect != sum {
		t.Errorf("empty preview checksum mismatch. expected: %s, got: %s", expect, sum)
	}

	p = Preview(&dataset.Dataset{Name: "a", Peername: "b", Path: "c"})

	expect = "747373b09aed281b2cbdb3655fa19dcd277ae3a5"
	sum = dstest.DatasetChecksum(p)
	if expect != sum {
		t.Errorf("preview with ref details mismatch. expected: %s, got: %s", expect, sum)
	}
}
