package cmd

import (
	"context"
	"testing"

	"github.com/qri-io/qri/errors"
)

func TestFullFieldToAbbr(t *testing.T) {
	cases := []struct {
		field, exp string
	}{
		{"commit", "cm"},
		{"structure", "st"},
		{"body", "bd"},
		{"meta", "md"},
		{"viz", "vz"},
		{"transform", "tf"},
		{"rendered", "rd"},
		{"foo", "foo"},
	}

	for i, c := range cases {
		got := fullFieldToAbbr(c.field)
		if got != c.exp {
			t.Errorf("case %d, for field '%s', expected '%s'. got '%s'", i, c.field, c.exp, got)
		}
	}
}

func TestAbbrFieldToFull(t *testing.T) {
	cases := []struct {
		field, exp string
	}{
		{"cm", "commit"},
		{"st", "structure"},
		{"bd", "body"},
		{"md", "meta"},
		{"vz", "viz"},
		{"tf", "transform"},
		{"rd", "rendered"},
		{"foo", "foo"},
	}

	for i, c := range cases {
		got := abbrFieldToFull(c.field)
		if got != c.exp {
			t.Errorf("case %d, for field '%s', expected '%s'. got '%s'", i, c.field, c.exp, got)
		}
	}
}

func TestDAGComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer_dag_complete", "qri_test_dag_complete")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args, expRefs []string
		label, err    string
	}{
		{[]string{}, []string{}, "", ""},
		{[]string{"dataset/ref"}, []string{"dataset/ref"}, "", ""},
		{[]string{"bad_label dataset/ref"}, []string{"bad_label dataset/ref"}, "", ""},
		{[]string{"meta"}, []string{}, "md", ""},
		{[]string{"structure", "dataset/ref"}, []string{"dataset/ref"}, "st", ""},
		{[]string{"vz", "dataset/ref"}, []string{"dataset/ref"}, "vz", ""},
	}
	for i, c := range cases {
		opt := &DAGOptions{
			IOStreams: run.Streams,
		}

		opt.Complete(f, c.args, true)
		if c.err != run.ErrStream.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, run.ErrStream.String())
			run.IOReset()
			continue
		}

		if opt.inst == nil {
			t.Errorf("case %d, opt.inst not set.", i)
		}

		if opt.Label != c.label {
			t.Errorf("case %d, label mismatch. Expected: '%s', Got: '%s'", i, c.label, opt.Label)
		}

		if len(opt.Refs) != len(c.expRefs) {
			t.Errorf("case %d, expected Refs mismatch. Expected: %s, Got: %s", i, c.expRefs, opt.Refs)
			run.IOReset()
			continue
		}

		for i, ref := range c.expRefs {
			if opt.Refs[i] != ref {
				t.Errorf("case %d, expected Refs mismatch. Expected: %s, Got: %s", i, c.expRefs, opt.Refs)
				break
			}
		}
		run.IOReset()
	}
}

func TestDAGInfo(t *testing.T) {
	run := NewTestRunner(t, "test_peer_dag_info", "qri_test_dag_info")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		description string
		opt         *DAGOptions
		stdout      string
		err         string
		errMsg      string
	}{
		{"dag with no options",
			&DAGOptions{},
			"",
			"dataset reference required",
			"",
		},
		// TODO (ramfox): blocked. Need a test subpackage in the dag package
		// that mocks the NodeGetter for testing purposes
		// {"dag info at a reference",
		// 	&DAGOptions{Refs: []string{"me/movies"}},
		// 	"0 elements. 0 insxrts. 0 deletes. 1 update.\n\n~ title: \"example city data\"\n",
		// 	"", "",
		// },
		//     {"diff json output",
		//       &DiffOptions{Left: "me/movies", Right: "me/cities", Selector: "meta", Format: "json"},
		//       `[{"type":"update","path":"/title","value":"example city data","originalValue":"example movie data"}]
		// `,
		//       "", "",
		//     },
	}
	for i, c := range cases {
		inst, err := f.Instance()
		if err != nil {
			t.Errorf("case %d, error creating inst: %s", i, err)
			continue
		}

		opt := c.opt
		opt.IOStreams = run.Streams
		opt.inst = inst

		err = opt.Info()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
			run.IOReset()
			continue
		}

		if libErr, ok := err.(errors.Error); ok {
			if libErr.Message() != c.errMsg {
				t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: '%s'", i, c.errMsg, libErr.Message())
				run.IOReset()
				continue
			}
		} else if c.errMsg != "" {
			t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: ''", i, c.errMsg)
			run.IOReset()
			continue
		}

		if c.stdout != run.OutStream.String() {
			t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.stdout, run.OutStream.String())
			run.IOReset()
			continue
		}
		run.IOReset()
	}
}
