package cmd

import (
	"testing"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
)

func TestUpdateComplete(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(nil)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args []string
		ref  string
		err  string
	}{
		{[]string{}, "", ""},
		{[]string{"me/from"}, "me/from", ""},
		{[]string{"me/from", "me/to"}, "", ""},
	}

	for i, c := range cases {
		opt := &UpdateOptions{
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if c.ref != opt.Ref {
			t.Errorf("case %d, opt.From not set correctly. Expected: '%s', Got: '%s'", i, c.ref, opt.Ref)
			ioReset(in, out, errs)
			continue
		}

		if opt.updateMethods == nil {
			t.Errorf("case %d, opt.updateMethods not set.", i)
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}
}

func TestUpdateRun(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(nil)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		opt              *UpdateOptions
		expect, msg, err string
	}{
		// TODO (b5): fix tests
		// {&UpdateOptions{Ref: "other/bad_ref"}, "", "", "unknown dataset 'other/bad_ref'. please add before updating"},
		// {&UpdateOptions{Ref: "me/bad_ref"}, "", "", "unknown dataset 'peer/bad_ref'. please add before updating"},
	}

	for i, c := range cases {
		ioReset(in, out, errs)
		// dsr, err := f.DatasetRequests()
		// if err != nil {
		// 	t.Errorf("case %d, error creating dataset request: %s", i, err)
		// 	continue
		// }

		c.opt.IOStreams = streams
		c.opt.updateMethods = lib.NewUpdateMethods(f.Instance())

		// TODO (b5): fix tests
		err = c.opt.RunUpdate(nil)
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
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

		if c.expect != out.String() {
			t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.expect, out.String())
			continue
		}
	}
}
