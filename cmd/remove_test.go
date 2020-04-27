package cmd

import (
	"testing"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
)

func TestRemoveComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_remove_complete")
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
		opt := &RemoveOptions{
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

func TestRemoveValidate(t *testing.T) {
	cases := []struct {
		args []string
		err  string
		msg  string
	}{
		{[]string{}, lib.ErrBadArgs.Error(), "please specify a dataset path or name you would like to remove from your qri node"},
		{[]string{"me/test"}, "", ""},
		{[]string{"me/test", "me/test2"}, "", ""},
	}
	for i, c := range cases {
		opt := &RemoveOptions{
			Refs: NewListOfRefSelects(c.args),
		}

		err := opt.Validate()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: %s, Got: %s", i, c.err, err)
			continue
		}
		if libErr, ok := err.(errors.Error); ok {
			if libErr.Message() != c.msg {
				t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: '%s'", i, c.msg, libErr.Message())
				continue
			}
		} else if c.msg != "" {
			t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: ''", i, c.msg)
			continue
		}
	}
}

func TestRemoveRun(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_dag_info")
	defer run.Delete()

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args     []string
		revision int
		expected string
		err      string
		msg      string
	}{
		{[]string{}, -1, "", "repo: empty dataset reference", ""},
		{[]string{"me/bad_dataset"}, -1, "", "repo: not found", "could not find dataset 'me/bad_dataset'"},
		{[]string{"me/movies"}, -1, "removed entire dataset 'peer/movies@/map/QmZ6FC5pfDUB8CpbyoXhZzEJpHC8w675JE471WUg4FmAdo'\n", "", ""},
		{[]string{"me/cities", "me/counter"}, -1, "removed entire dataset 'peer/cities@/map/QmSEjjiLYtkQ8iy8icK4hKwq37JVDFXVtL21eGuEjPv6uA'\n", "", ""},
		{[]string{"me/movies"}, -1, "", "repo: not found", "could not find dataset 'me/movies'"},
	}

	for i, c := range cases {
		dsm, err := f.DatasetMethods()
		if err != nil {
			t.Errorf("case %d, error creating dataset request: %s", i, err)
			continue
		}

		opt := &RemoveOptions{
			IOStreams:      run.Streams,
			Refs:           NewListOfRefSelects(c.args),
			Revision:       dsref.Rev{Field: "ds", Gen: c.revision},
			DatasetMethods: dsm,
		}

		err = opt.Run()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
			run.IOReset()
			continue
		}

		if libErr, ok := err.(errors.Error); ok {
			if libErr.Message() != c.msg {
				t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: '%s'", i, c.msg, libErr.Message())
				run.IOReset()
				continue
			}
		} else if c.msg != "" {
			t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: ''", i, c.msg)
			run.IOReset()
			continue
		}

		if c.expected != run.OutStream.String() {
			t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.expected, run.OutStream.String())
			run.IOReset()
			continue
		}
		run.IOReset()
	}
}

func testSliceEqual(a, b []string) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
