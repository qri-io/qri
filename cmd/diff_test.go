package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDiffComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_diff_complete")
	defer run.Delete()

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args []string
		err  string
	}{
		{[]string{}, ""},
		{[]string{"one arg"}, ""},
		{[]string{"one arg", "two args"}, ""},
	}

	for i, c := range cases {
		opt := &DiffOptions{
			IOStreams: run.Streams,
		}

		opt.Complete(f, c.args)

		if c.err != run.ErrStream.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, run.ErrStream.String())
			run.IOReset()
			continue
		}

		if opt.DatasetMethods == nil {
			t.Errorf("case %d, opt.DatasetMethods not set.", i)
			run.IOReset()
			continue
		}
		run.IOReset()
	}
}

func TestDiffRun(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_dag_info")
	defer run.Delete()

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	good := []struct {
		description string
		opt         *DiffOptions
		stdout      string
	}{
		{"diff two dataset metas",
			&DiffOptions{
				Refs:     NewListOfRefSelects([]string{"me/movies", "me/cities"}),
				Selector: "meta",
			},
			`0 elements. 1 insert. 1 delete.

 qri: "md:0"
-title: "example movie data"
+title: "example city data"
`,
		},
		{"diff json output",
			&DiffOptions{
				Refs:     NewListOfRefSelects([]string{"me/movies", "me/cities"}),
				Selector: "meta",
				Format:   "json",
			},
			`{"stat":{"leftNodes":3,"rightNodes":3,"leftWeight":45,"rightWeight":43,"inserts":1,"deletes":1},"diff":[[" ","qri","md:0"],["-","title","example movie data"],["+","title","example city data"]]}
`,
		},
	}

	for _, c := range good {
		t.Run(c.description, func(t *testing.T) {
			dsm, err := f.DatasetMethods()
			if err != nil {
				t.Fatalf("case %s, error creating dataset request: %s", c.description, err)
			}

			opt := c.opt
			opt.IOStreams = run.Streams
			opt.DatasetMethods = dsm

			if err = opt.Run(); err != nil {
				t.Fatalf("case %s unexpected error: %s", c.description, err)
			}

			if diff := cmp.Diff(run.OutStream.String(), c.stdout); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}

			run.IOReset()
		})
	}

	bad := []struct {
		opt *DiffOptions
		err string
	}{
		{
			&DiffOptions{},
			"nothing to diff",
		},
	}

	for _, c := range bad {
		dsm, err := f.DatasetMethods()
		if err != nil {
			t.Errorf("case %s, error creating dataset request: %s", c.err, err)
			continue
		}

		opt := c.opt
		opt.Refs = NewListOfRefSelects([]string{})
		opt.IOStreams = run.Streams
		opt.DatasetMethods = dsm

		err = opt.Run()

		if err == nil {
			t.Errorf("expected: '%s', got no error", c.err)
			run.IOReset()
			continue
		}
		if c.err != err.Error() {
			t.Errorf("error mismatch. expected: '%s', got: '%s'", c.err, err.Error())
		}
		run.IOReset()
	}
}
