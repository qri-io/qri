package base

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
)

func TestRecall(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addNowTransformDataset(t, r)

	_, err := Recall(ctx, r.Filesystem(), ref, "")
	if err != nil {
		t.Error(err)
	}

	_, err = Recall(ctx, r.Filesystem(), ref, "tf")
	if err != nil {
		t.Error(err)
	}
}

func TestLoadRevisions(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	cities, err := dsfs.LoadDataset(ctx, r.Filesystem(), ref.Path)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		ref  dsref.Ref
		revs string
		ds   *dataset.Dataset
		err  string
	}{
		// TODO - both of these are failing, let's make 'em work:
		// "ds" Qri value is mismatching
		// {ref, "ds", cities, ""},
		// Logic on what to do in "body" is a little confusing atm, do we set BodyPath
		// and move on?
		// {ref, "bd", cities, ""},

		{ref, "tf", &dataset.Dataset{Transform: cities.Transform}, ""},
		{ref, "md,vz,tf,st", &dataset.Dataset{Transform: cities.Transform, Meta: cities.Meta, Structure: cities.Structure}, ""},
	}

	for i, c := range cases {
		revs, err := dsref.ParseRevs(c.revs)
		if err != nil {
			t.Errorf("case %d error parsing revs: %s", i, err)
			continue
		}

		got, err := LoadRevs(ctx, r.Store(), c.ref, revs)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
		}

		if err := dataset.CompareDatasets(c.ds, got); err != nil {
			t.Errorf("case %d result mismatch: %s", i, err)
		}
	}
}

func TestDrop(t *testing.T) {
	good := []struct {
		str string
		ds  *dataset.Dataset
	}{
		{"md,st,bd,tf,rm,vz", &dataset.Dataset{}},

		{"md", &dataset.Dataset{Meta: &dataset.Meta{}}},
		{"meta", &dataset.Dataset{Meta: &dataset.Meta{}}},
		{"st", &dataset.Dataset{Structure: &dataset.Structure{}}},
		{"structure", &dataset.Dataset{Structure: &dataset.Structure{}}},
	}

	expect := &dataset.Dataset{}
	for _, c := range good {
		t.Run(c.str, func(t *testing.T) {
			if err := Drop(c.ds, c.str); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expect, c.ds, cmpopts.IgnoreUnexported(dataset.Dataset{})); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}

	bad := []struct {
		str, err string
	}{
		{"ds", `cannot drop component: "ds"`},
		{"snarfar", `unrecognized revision field: snarfar`},
	}

	for _, c := range bad {
		t.Run(c.str, func(t *testing.T) {
			err := Drop(&dataset.Dataset{}, c.str)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if diff := cmp.Diff(c.err, err.Error(), cmpopts.IgnoreUnexported(dataset.Dataset{})); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
