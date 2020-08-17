package cmd

import (
	"context"
	"testing"

	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
)

func TestRenameComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer_rename_complete", "qri_test_rename_complete")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args       []string
		expectFrom string
		expectTo   string
		err        string
	}{
		{[]string{}, "", "", ""},
		{[]string{"me/from"}, "", "", ""},
		{[]string{"me/from", "me/to"}, "me/from", "me/to", ""},
	}

	for i, c := range cases {
		opt := &RenameOptions{
			IOStreams: run.Streams,
		}

		opt.Complete(f, c.args)

		if c.err != run.ErrStream.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, run.ErrStream.String())
			run.IOReset()
			continue
		}

		if c.expectFrom != opt.From {
			t.Errorf("case %d, opt.From not set correctly. Expected: '%s', Got: '%s'", i, c.expectFrom, opt.From)
			run.IOReset()
			continue
		}

		if c.expectTo != opt.To {
			t.Errorf("case %d, opt.From not set correctly. Expected: '%s', Got: '%s'", i, c.expectTo, opt.To)
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

func TestRenameValidate(t *testing.T) {
	cases := []struct {
		from string
		to   string
		err  string
		msg  string
	}{
		{"", "", lib.ErrBadArgs.Error(), "please provide two dataset names, the original and the new name, for example:\n    $ qri rename me/old_name me/new_name\nsee `qri rename --help` for more details"},
		{"me/from", "", lib.ErrBadArgs.Error(), "please provide two dataset names, the original and the new name, for example:\n    $ qri rename me/old_name me/new_name\nsee `qri rename --help` for more details"},
		{"", "me/to", lib.ErrBadArgs.Error(), "please provide two dataset names, the original and the new name, for example:\n    $ qri rename me/old_name me/new_name\nsee `qri rename --help` for more details"},
		{"me/from", "me/to", "", ""},
	}
	for i, c := range cases {
		opt := &RenameOptions{
			From: c.from,
			To:   c.to,
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

func TestRenameRun(t *testing.T) {
	run := NewTestRunner(t, "test_peer_rename_run", "qri_test_rename_run")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		from     string
		to       string
		expected string
		err      string
		msg      string
	}{
		{"", "", "", "current name is required to rename a dataset", ""},
		{"me/from", "", "", "reference not found", ""},
		{"", "me/to", "", "current name is required to rename a dataset", ""},
		{"me/bad_name", "me/bad_name_too", "", "reference not found", ""},
		{"me/cities", "me/cities_too", "renamed dataset to cities_too\n", "", ""},
	}

	for i, c := range cases {
		dsm, err := f.DatasetMethods()
		if err != nil {
			t.Errorf("case %d, error creating dataset request: %s", i, err)
			continue
		}

		opt := &RenameOptions{
			IOStreams:      run.Streams,
			From:           c.from,
			To:             c.to,
			DatasetMethods: dsm,
		}

		err = opt.Run()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: %q, Got: %q", i, c.err, err)
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
