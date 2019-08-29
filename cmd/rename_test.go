package cmd

import (
	"testing"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
)

func TestRenameComplete(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory()
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
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if c.expectFrom != opt.From {
			t.Errorf("case %d, opt.From not set correctly. Expected: '%s', Got: '%s'", i, c.expectFrom, opt.From)
			ioReset(in, out, errs)
			continue
		}

		if c.expectTo != opt.To {
			t.Errorf("case %d, opt.From not set correctly. Expected: '%s', Got: '%s'", i, c.expectTo, opt.To)
			ioReset(in, out, errs)
			continue
		}

		if opt.DatasetRequests == nil {
			t.Errorf("case %d, opt.DatasetRequests not set.", i)
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
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
		if libErr, ok := err.(lib.Error); ok {
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
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory()
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
		{"", "", "", "repo: empty dataset reference", ""},
		{"me/from", "", "", "repo: empty dataset reference", ""},
		{"", "me/to", "", "repo: empty dataset reference", ""},
		{"me/bad_name", "me/bad_name_too", "", "error with existing reference: repo: not found", ""},
		{"me/cities", "me/cities_too", "renamed dataset cities_too\n", "", ""},
	}

	for i, c := range cases {
		dsr, err := f.DatasetRequests()
		if err != nil {
			t.Errorf("case %d, error creating dataset request: %s", i, err)
			continue
		}

		opt := &RenameOptions{
			IOStreams:       streams,
			From:            c.from,
			To:              c.to,
			DatasetRequests: dsr,
		}

		err = opt.Run()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
			ioReset(in, out, errs)
			continue
		}

		if libErr, ok := err.(lib.Error); ok {
			if libErr.Message() != c.msg {
				t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: '%s'", i, c.msg, libErr.Message())
				ioReset(in, out, errs)
				continue
			}
		} else if c.msg != "" {
			t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: ''", i, c.msg)
			ioReset(in, out, errs)
			continue
		}

		if c.expected != out.String() {
			t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.expected, out.String())
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}
}
