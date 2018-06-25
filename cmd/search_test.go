package cmd

import (
	"testing"

	"github.com/qri-io/qri/lib"
)

func TestSearchComplete(t *testing.T) {
	streams, in, out, errs := NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args   []string
		expect string
		err    string
	}{
		{[]string{}, "", ""},
		{[]string{"test"}, "test", ""},
		{[]string{"test", "test2"}, "test", ""},
	}

	for i, c := range cases {
		opt := &SearchOptions{
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if c.expect != opt.Query {
			t.Errorf("case %d, opt.Ref not set correctly. Expected: '%s', Got: '%s'", i, c.expect, opt.Query)
			ioReset(in, out, errs)
			continue
		}

		if opt.SearchRequests == nil {
			t.Errorf("case %d, opt.SearchRequests not set.", i)
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}
}

func TestSearchValidate(t *testing.T) {
	cases := []struct {
		query  string
		err    string
		errMsg string
	}{
		{"test", "", ""},
		{"", ErrBadArgs.Error(), "please provide search parameters, for example:\n    $ qri search census\n    $ qri search 'census 2018'\nsee `qri search --help` for more information"},
	}
	for i, c := range cases {
		opt := &SearchOptions{
			Query: c.query,
		}

		err := opt.Validate()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: %s, Got: %s", i, c.err, err)
			continue
		}
		if libErr, ok := err.(lib.Error); ok {
			if libErr.Message() != c.errMsg {
				t.Errorf("case %d, mismatched user-friendly message. Expected: %s, Got: %s", i, c.errMsg, libErr.Message())
				continue
			}
		}
	}
}

// func TestSearchRun(t *testing.T) {
// 	streams, in, out, errs := NewTestIOStreams()
// 	setNoColor(true)

// 	f, err := NewTestFactory()
// 	if err != nil {
// 		t.Errorf("error creating new test factory: %s", err)
// 		return
// 	}

// 	cases := []struct {
// 		query    string
// 		format   string
// 		expected string
// 		err      string
// 		msg      string
// 	}{
// 		{"movies", "", "", "", ""},
// 	}

// 	for i, c := range cases {
// 		sr, err := f.SearchRequests()
// 		if err != nil {
// 			t.Errorf("case %d, error creating dataset request: %s", i, err)
// 			continue
// 		}

// 		opt := &SearchOptions{
// 			IOStreams:      streams,
// 			Query:          c.query,
// 			Format:         c.format,
// 			SearchRequests: sr,
// 		}

// 		err = opt.Run()

// 		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
// 			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
// 			ioReset(in, out, errs)
// 			continue
// 		}

// 		if libErr, ok := err.(lib.Error); ok {
// 			if libErr.Message() != c.msg {
// 				t.Errorf("case %d, mismatched user-friendly error. Expected: '%s', Got: '%v'", i, c.msg, libErr.Message())
// 				ioReset(in, out, errs)
// 				continue
// 			}
// 		}

// 		if c.expected != out.String() {
// 			t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.expected, out.String())
// 			ioReset(in, out, errs)
// 			continue
// 		}
// 		ioReset(in, out, errs)
// 	}
// }
