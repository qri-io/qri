package stats

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi"
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

func TestStatsFSI(t *testing.T) {
	ctx := context.Background()

	workDir, err := ioutil.TempDir("", "qri_test_stats_service_fsi")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)

	mr, err := repotest.NewTestRepo()
	if err != nil {
		t.Fatal(err)
	}

	cache, err := NewLocalCache(workDir, 1000<<8)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(cache)

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

	fsiDir := filepath.Join(workDir, "fsi_link")
	if err := os.MkdirAll(fsiDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	fsiSvc := fsi.NewFSI(mr, nil)
	vi, _, err := fsiSvc.CreateLink(ctx, fsiDir, ref)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(vi)
	if err = fsi.WriteComponents(ds, fsiDir, mr.Filesystem()); err != nil {
		t.Fatal(err)
	}

	if ds, err = fsi.ReadDir(fsiDir); err != nil {
		t.Fatal(err)
	}
	if err = base.OpenDataset(ctx, mr.Filesystem(), ds); err != nil {
		t.Fatal(err)
	}

	sa, err := svc.Stats(ctx, ds)
	if err != nil {
		t.Fatal(err)
	}

	// alter FSI body file
	const alteredBodyData = `city,pop,avg_age,in_usa
	toronto,4000000,55.0,false
	new york,850000,44.0,false
	chicago,30000,440.4,false
	chatham,3500,650.25,false
	raleigh,25000,5000.65,false`

	if err := ioutil.WriteFile(filepath.Join(fsiDir, "body.csv"), []byte(alteredBodyData), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if ds, err = fsi.ReadDir(fsiDir); err != nil {
		t.Fatal(err)
	}
	if err = base.OpenDataset(ctx, mr.Filesystem(), ds); err != nil {
		t.Fatal(err)
	}

	updatedSa, err := svc.Stats(ctx, ds)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(sa, updatedSa); diff == "" {
		t.Errorf("expected stats to change with body file to change. found no changes")
	}
}
