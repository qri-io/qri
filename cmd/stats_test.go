package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStatsComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_stats_complete")
	defer run.Delete()

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	badCases := []struct {
		description string
		args        []string
		err         string
	}{
		{"no args", []string{}, "repo: empty dataset reference"},
	}

	for i, c := range badCases {
		opt := &StatsOptions{
			IOStreams: run.Streams,
		}

		err := opt.Complete(f, c.args)

		if c.err != err.Error() {
			t.Errorf("%d. case %s, error mismatch. Expected: '%s', Got: '%s'", i, c.description, c.err, err.Error())
			run.IOReset()
			continue
		}

		run.IOReset()
	}

	goodCases := []struct {
		description string
		args        []string
		expectedRef string
	}{
		{"given one ref", []string{"me/ref"}, "me/ref"},
		{"given multiple refs", []string{"me/ref", "me/ref_foo"}, "me/ref"},
	}

	for i, c := range goodCases {
		opt := &StatsOptions{
			IOStreams: run.Streams,
		}

		if err := opt.Complete(f, c.args); err != nil {
			t.Errorf("%d. case %s, unexpected error: '%s'", i, c.description, err.Error())
		}

		if c.expectedRef != opt.Refs.Ref() {
			t.Errorf("%d. case %s, incorrect ref. Expected: '%s', Got: '%s'", i, c.description, c.expectedRef, opt.Refs.Ref())
			run.IOReset()
			continue
		}

		if opt.DatasetRequests == nil {
			t.Errorf("%d. case %s, opt.DatasetRequests not set.", i, c.description)
			run.IOReset()
			continue
		}
		run.IOReset()
	}
}

func TestStatsRun(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_stats_run")
	defer run.Delete()

	f, err := NewTestFactory()
	if err != nil {
		t.Fatalf("error creating new test factory: %s", err)
	}

	dsr, err := f.DatasetRequests()
	if err != nil {
		t.Fatalf("error creating dataset request: %s", err)
	}
	badCases := []struct {
		description string
		ref         string
		err         string
	}{
		{"empty ref", "", "repo: empty dataset reference"},
		{"dataset does not exist", "me/dataset_does_not_exist", "repo: not found"},
	}

	for i, c := range badCases {
		run.IOReset()

		opt := &StatsOptions{
			IOStreams:       run.Streams,
			Refs:            NewExplicitRefSelect(c.ref),
			DatasetRequests: dsr,
		}

		if err = opt.Run(); err.Error() != c.err {
			t.Errorf("%d. case '%s', error mismatch, expected: %s, got: %s ", i, c.description, c.err, err)
			continue
		}
	}

	goodCases := []struct {
		description string
		ref         string
		// TODO (ramfox): this is cheating, we should check that the out stream is
		// getting the proper output. This will be more important when we have
		// nicely formatted stats. The veracity of the stats are being checked lower
		// in the stack
	}{
		{"csv", "me/cities"},
		{"json", "me/sitemap"},
	}

	for i, c := range goodCases {
		run.IOReset()

		opt := &StatsOptions{
			IOStreams:       run.Streams,
			Refs:            NewExplicitRefSelect(c.ref),
			DatasetRequests: dsr,
		}

		if err = opt.Run(); err != nil {
			t.Errorf("%d. case '%s', unexpected error: %s ", i, c.description, err)
			continue
		}
	}
}

func TestStatsFSI(t *testing.T) {
	run := NewFSITestRunner(t, "qri_test_stats_fsi")
	defer run.Delete()

	run.CreateAndChdirToWorkDir("stats_fsi")

	// Init as a linked directory.
	run.MustExec(t, "qri init --name move_dir --format csv")
	// Save the new dataset.
	run.MustExec(t, "qri save")

	output := run.MustExecCombinedOutErr(t, "qri stats")

	expect := `for linked dataset [test_peer/move_dir]

[{"count":2,"maxLength":4,"minLength":3,"type":"string","unique":2},{"count":2,"maxLength":4,"minLength":3,"type":"string","unique":2},{"count":2,"histogram":{"bins":[3,3.4,3.8,4.2,4.6,5,5.4,5.800000000000001,6.2,6.6,7],"frequencies":[1,0,0,0,0,0,0,1,0,0]},"max":6,"mean":4.5,"median":4.5,"min":3,"type":"numeric"}]

`

    if diff := cmp.Diff(expect, output); diff != "" {
        t.Errorf("output mismatch (-want +got):\n%s", diff)
    }

    output = run.MustExecCombinedOutErr(t, "qri stats --pretty")

    expect = `for linked dataset [test_peer/move_dir]

[
  {
    "count": 2,
    "maxLength": 4,
    "minLength": 3,
    "type": "string",
    "unique": 2
  },
  {
    "count": 2,
    "maxLength": 4,
    "minLength": 3,
    "type": "string",
    "unique": 2
  },
  {
    "count": 2,
    "histogram": {
      "bins": [
        3,
        3.4,
        3.8,
        4.2,
        4.6,
        5,
        5.4,
        5.800000000000001,
        6.2,
        6.6,
        7
      ],
      "frequencies": [
        1,
        0,
        0,
        0,
        0,
        0,
        0,
        1,
        0,
        0
      ]
    },
    "max": 6,
    "mean": 4.5,
    "median": 4.5,
    "min": 3,
    "type": "numeric"
  }
]
`

    if diff := cmp.Diff(expect, output); diff != "" {
        t.Errorf("output mismatch (-want +got):\n%s", diff)
    }
}
