package cmd

import (
	"testing"

	"github.com/qri-io/qri/lib"
)

func TestSaveComplete(t *testing.T) {
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
		opt := &SaveOptions{
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if c.expect != opt.Ref {
			t.Errorf("case %d, opt.Ref not set correctly. Expected: '%s', Got: '%s'", i, c.expect, opt.Ref)
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

func TestSaveValidate(t *testing.T) {
	cases := []struct {
		ref      string
		filepath string
		bodypath string
		err      string
		msg      string
	}{
		{"", "", "", ErrBadArgs.Error(), "please provide the peername and dataset name you would like to update, in the format of `peername/dataset_name`\nsee `qri save --help` for more info"},
		{"me/test", "", "", ErrBadArgs.Error(), "please an updated/changed dataset file (--file) or body file (--body), or both\nsee `qri save --help` for more info"},
		{"me/test", "test/path.yaml", "", "", ""},
		{"me/test", "", "test/bodypath.yaml", "", ""},
		{"me/test", "test/filepath.yaml", "test/bodypath.yaml", "", ""},
	}
	for i, c := range cases {
		opt := &SaveOptions{
			Ref:      c.ref,
			FilePath: c.filepath,
			BodyPath: c.bodypath,
		}

		err := opt.Validate()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%s'", i, c.err, err)
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

// func TestSaveRun(t *testing.T) {
//  streams, in, out, errs := NewTestIOStreams()
//  setNoColor(true)
//  setNoPrompt(true)

//  f, err := NewTestFactory()
//  if err != nil {
//    t.Errorf("error creating new test factory: %s", err)
//    return
//  }

//  cases := []struct {
//    ref        string
//    filepath   string
//    bodypath   string
//    title      string
//    message    string
//    noRegistry bool
//    secrets    []string
//  }{
//    {""},
//  }

//  for i, c := range cases {
//    dsr, err := f.DatasetRequests()
//    if err != nil {
//      t.Errorf("case %d, error creating dataset request: %s", i, err)
//      continue
//    }

//    opt := &SaveOptions{
//      IOStreams:       streams,
//      Ref:             c.ref,
//      FilePath:        c.filepath,
//      BodyPath:        c.bodypath,
//      Title:           c.title,
//      Message:         c.message,
//      NoRegistry:      c.noRegistry,
//      Secrets:         c.secrets,
//      DatasetRequests: dsr,
//    }

//    err = opt.Run()
//    if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
//      t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
//      ioReset(in, out, errs)
//      continue
//    }

//    if libErr, ok := err.(lib.Error); ok {
//      if libErr.Message() != c.msg {
//        t.Errorf("case %d, mismatched user-friendly error. Expected: '%s', Got: '%v'", i, c.msg, libErr.Message())
//        ioReset(in, out, errs)
//        continue
//      }
//    }

//    if c.expected != out.String() {
//      t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.expected, out.String())
//      ioReset(in, out, errs)
//      continue
//    }
//    ioReset(in, out, errs)
//  }
// }
