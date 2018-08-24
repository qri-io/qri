package cmd

import (
	"testing"

	"github.com/qri-io/qri/lib"
)

func TestValidateComplete(t *testing.T) {
	// in, out, and errs are buffers
	streams, in, out, errs := NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(nil)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args           []string
		filepath       string
		schemaFilepath string
		expect         string
		err            string
	}{
		{[]string{}, "filepath", "schemafilepath", "", ""},
		{[]string{"test"}, "", "", "test", ""},
		{[]string{"foo", "bar"}, "", "", "foo", ""},
	}

	for i, c := range cases {
		opt := &ValidateOptions{
			IOStreams:      streams,
			Filepath:       c.filepath,
			SchemaFilepath: c.schemaFilepath,
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

func TestValidateRun(t *testing.T) {
	streams, in, out, errs := NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(nil)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	path, ok := currentPath()
	if !ok {
		t.Errorf("error getting path to current folder")
		return
	}

	cases := []struct {
		ref            string
		filePath       string
		schemaFilePath string
		url            string
		expected       string
		err            string
		msg            string
	}{
		{"", "", "", "", "", "bad arguments provided", "please provide a dataset name, or a supply the --body and --schema flags with file paths"},
		// TODO: add back when we again support validating from a URL
		// {"", "", "", "url", "", "bad arguments provided", "if you are validating data from a url, please include a dataset name or supply the --schema flag with a file path that Qri can validate against"},
		{"peer/movies", "", "", "", movieOutput, "", ""},
		{"peer/bad_dataset", "", "", "", "", "cannot find dataset: peer/bad_dataset@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", ""},
		{"", "bad/filepath", "testdata/days_of_week_schema.json", "", "", "open " + path + "/bad/filepath: no such file or directory", "error opening body file: could not open " + path + "/bad/filepath: no such file or directory"},
		{"", "testdata/days_of_week.csv", "bad/schema_filepath", "", "", "open " + path + "/bad/schema_filepath: no such file or directory", "error opening schema file: could not open " + path + "/bad/schema_filepath: no such file or directory"},
		{"", "testdata/days_of_week.csv", "testdata/days_of_week_schema.json", "", "✔ All good!\n", "", ""},
		// TODO: pull from url
	}

	for i, c := range cases {
		dsr, err := f.DatasetRequests()
		if err != nil {
			t.Errorf("case %d, error creating dataset request: %s", i, err)
			continue
		}

		opt := &ValidateOptions{
			IOStreams:       streams,
			Ref:             c.ref,
			Filepath:        c.filePath,
			SchemaFilepath:  c.schemaFilePath,
			URL:             c.url,
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

var movieOutput = `0: /4/1: "" type should be integer
1: /199/1: "" type should be integer
2: /206/1: "" type should be integer
3: /1510/1: "" type should be integer
4: /3604/1: "" type should be integer
5: /3815/1: "" type should be integer
6: /3834/1: "" type should be integer
7: /4299/1: "" type should be integer
8: /4392/1: "" type should be integer
9: /4397/1: "" type should be integer
10: /4517/1: "" type should be integer
11: /4609/1: "" type should be integer
12: /4690/1: "" type should be integer
13: /4948/1: "" type should be integer
14: /4989/1: "" type should be integer
`
