package cmd

import (
	"bytes"
	// "io/ioutil"
	"testing"
)

func TestValidateComplete(t *testing.T) {
	// in, out, and errs are buffers
	streams, in, out, errs := NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(streams)
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
		{[]string{}, "", "", "", "you need to provide a dataset name or a supply the --data and --schema flags with file paths\n"},
		{[]string{}, "filepath", "schemafilepath", "", ""},
		{[]string{"test"}, "", "", "test", ""},
		{[]string{"foo", "bar"}, "", "", "foo", ""},
	}

	for i, c := range cases {
		opt := &ValidateOptions{
			IOStreams:      f.IOStreams,
			Filepath:       c.filepath,
			SchemaFilepath: c.schemaFilepath,
		}
		opt.Complete(f, c.args)

		if errs.String() != c.err {
			t.Errorf("case %v, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if opt.Ref != c.expect {
			t.Errorf("case %v, opt.Ref not set correctly. Expected: '%s', Got: '%s'", i, c.expect, opt.Ref)
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}

}

func TestValidateRun(t *testing.T) {
	streams, in, out, errs := NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(streams)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		ref            string
		filePath       string
		schemaFilePath string
		url            string
		expected       string
		err            string
	}{
		{"peer/movies", "", "", "", movieOutput, ""},
		{"peer/bad_dataset", "", "", "", "", "cannot find dataset: peer/bad_dataset@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"},
		{"", "bad/filepath", "testdata/days_of_week_schema.json", "", "", "open /Users/ramfox/go/src/github.com/qri-io/qri/cmd/bad/filepath: no such file or directory"},
		{"", "testdata/days_of_week.csv", "bad/schema_filepath", "", "", "open /Users/ramfox/go/src/github.com/qri-io/qri/cmd/bad/schema_filepath: no such file or directory"},
		{"", "testdata/days_of_week.csv", "testdata/days_of_week_schema.json", "", "✔ All good!\n", ""},
		// TOD0: pull from url
	}

	for i, c := range cases {
		dsr, err := f.DatasetRequests()
		if err != nil {
			t.Errorf("case %v, error creating dataset request: %s", i, err)
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
			t.Errorf("case %v, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
			ioReset(in, out, errs)
			continue
		}
		if out.String() != c.expected {
			t.Errorf("case %v, output mismatch. Expected: '%s', Got: '%s'", i, c.expected, out.String())
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}
}

func ioReset(in, out, errs *bytes.Buffer) {
	in.Reset()
	out.Reset()
	errs.Reset()
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
