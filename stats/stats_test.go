package stats

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestStatsService(t *testing.T) {
	ctx := context.Background()

	workDir, err := ioutil.TempDir("", "qri_test_stats_service")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)

	mr, err := repotest.NewTestRepo()
	if err != nil {
		t.Fatal(err)
	}

	ref := dsref.MustParse("peer/cities")
	if _, err := mr.ResolveRef(ctx, &ref); err != nil {
		t.Fatal(err)
	}

	ds, err := dsfs.LoadDataset(ctx, mr.Filesystem(), ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err = base.OpenDataset(ctx, mr.Filesystem(), ds); err != nil {
		t.Fatal(err)
	}

	cache, err := NewLocalCache(workDir, 1000<<8)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(cache)

	expect := ds.Stats
	sa, err := svc.Stats(ctx, ds)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expect, sa); diff != "" {
		t.Errorf("stat component result mismatch. (-want +got):%s\n", diff)
	}

	// TODO(b5): dataset.Dataset.BodyFile() should return a name that matches.
	// currently we need to set the filename manually so detect.Structure has something
	// to work with
	ds.SetBodyFile(qfs.NewMemfileReader(ds.Structure.BodyFilename(), ds.BodyFile()))

	// remove path. recalculated stats won't have a path set
	expect.Path = ""
	// drop stats & structure to force recalculation of both
	ds.Stats = nil
	ds.Structure = nil

	sa, err = svc.Stats(ctx, ds)
	if err != nil {
		t.Fatal(err)
	}

	// TODO (b5) - there are currently discrepencies in the types created by
	// dsstats.ToMap and the types inferred by a trip through JSON serialization
	// so we need to marshal/unmarshal the result. The proper fix for this
	// is stronger typing on the Stats field of the Stats component
	data, err := json.Marshal(sa)
	if err != nil {
		t.Fatal(err)
	}
	sa = &dataset.Stats{}
	if err := json.Unmarshal(data, sa); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expect, sa); diff != "" {
		t.Errorf("calculated stat result mismatch. (-want +got):%s\n", diff)
	}

	// drop stats again. at this point the body file is consumed,
	// but we should hit the cache
	ds.Stats = nil
	ds.Structure = nil

	sa, err = svc.Stats(ctx, ds)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expect, sa); diff != "" {
		t.Errorf("cached stat result mismatch. (-want +got):%s\n", diff)
	}
}
