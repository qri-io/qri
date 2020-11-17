package base

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/base/dsfs"
)

func TestListDatasets(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	// Limit to one
	res, err := ListDatasets(ctx, r, "", 1, 0, false, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 1 {
		t.Error("expected one dataset response")
	}

	// Limit to published datasets
	res, err = ListDatasets(ctx, r, "", 1, 0, false, true, false)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 0 {
		t.Error("expected no published datasets")
	}

	if err := SetPublishStatus(r, ref, true); err != nil {
		t.Fatal(err)
	}

	// Limit to published datasets, after publishing cities
	res, err = ListDatasets(ctx, r, "", 1, 0, false, true, false)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 1 {
		t.Error("expected one published dataset response")
	}

	// Limit to datasets with "city" in their name
	res, err = ListDatasets(ctx, r, "city", 1, 0, false, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 0 {
		t.Error("expected no datasets with \"city\" in their name")
	}

	// Limit to datasets with "cit" in their name
	res, err = ListDatasets(ctx, r, "cit", 1, 0, false, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 1 {
		t.Error("expected one dataset with \"cit\" in their name")
	}
}

func TestRawDatasetRefs(t *testing.T) {
	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	minute := 0
	dsfs.Timestamp = func() time.Time {
		minute++
		return time.Date(2001, 01, 01, 01, minute, 01, 01, time.UTC)
	}

	ctx := context.Background()
	r := newTestRepo(t)
	addCitiesDataset(t, r)

	actual, err := RawDatasetRefs(ctx, r)
	if err != nil {
		t.Fatal(err)
	}

	expect := dstest.Template(t, `0 Peername:  peer
  ProfileID: {{ .ProfileID }}
  Name:      cities
  Path:      {{ .Path }}
  FSIPath:   
  Published: false
`, map[string]string{
		"ProfileID": "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		"Path":      "/mem/QmUfofLxWuGjECZcN1JbZaXZ2C2kQckpNArPeNutLpbuKE",
	})

	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
