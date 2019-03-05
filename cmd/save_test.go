package cmd

import (
	"testing"
	"time"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/registry/pinset"
	"github.com/qri-io/registry/regserver/mock"
)

func TestSaveComplete(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(nil)
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

func TestSaveRun(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)
	setNoPrompt(true)

	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	reg := mock.NewMemRegistry()
	ps := &pinset.MemPinset{Profiles: reg.Profiles}
	c, _ := mock.NewMockServerRegistryPinset(reg, ps)

	f, err := NewTestFactory(c)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	lib.Config = config.DefaultConfigForTesting()

	_, ok := currentPath()
	if !ok {
		t.Errorf("error getting path to current folder")
		return
	}

	cases := []struct {
		description string
		ref         string
		filepath    string
		bodypath    string
		title       string
		message     string
		publish     bool
		dryrun      bool
		expect      string
		err         string
		msg         string
	}{
		{"no data", "me/bad_dataset", "", "", "", "", false, false, "", "no changes to save", ""},

		{"bad dataset file", "me/cities", "bad/filpath.json", "", "", "", false, false, "", "open bad/filpath.json: no such file or directory", ""},

		{"bad body file", "me/cities", "", "bad/bodypath.csv", "", "", false, false, "", "opening dataset.bodyPath 'bad/bodypath.csv': path not found", ""},

		{"good inputs, dryrun", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_ten.csv", "", "", false, true, "dataset saved: peer/movies@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/QmUwGEwtP2B1Y2LqRQttwGSmvNEJQYXkoLba5JKjMp6uBb\nthis dataset has 1 validation errors\n", "", ""},

		{"good inputs", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_ten.csv", "", "", true, false, "dataset saved: peer/movies@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/QmUwGEwtP2B1Y2LqRQttwGSmvNEJQYXkoLba5JKjMp6uBb\nthis dataset has 1 validation errors\n", "", ""},

		{"add rows, dry run", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "Added 10 more rows", "Adding to the number of rows in dataset", false, true, "dataset saved: peer/movies@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/QmdKoLw1RFz8SK3GpGN7LBHi2hKVoDPnacJdQuWZmu6PBG\nthis dataset has 1 validation errors\n", "", ""},

		{"add rows, save", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "Added 10 more rows", "Adding to the number of rows in dataset", true, false, "dataset saved: peer/movies@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/QmdKoLw1RFz8SK3GpGN7LBHi2hKVoDPnacJdQuWZmu6PBG\nthis dataset has 1 validation errors\n", "", ""},

		{"no changes detected", "me/movies", "testdata/movies/dataset.json", "testdata/movies/body_twenty.csv", "trying to add again", "hopefully this errors", false, false, "", "error saving: no changes detected", ""},

		{"add viz", "me/movies", "testdata/movies/dataset_with_viz.json", "", "", "", false, false, "dataset saved: peer/movies@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/Qmcmjcw1onuvWCvM2NZzqDztfp3djpCErcD2zSirmAw65p\nthis dataset has 1 validation errors\n", "", ""},

		{"add transform", "me/movies", "testdata/movies/dataset_with_tf.json", "", "", "", false, false, "dataset saved: peer/movies@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/QmTwkFwddz8pFHj4z7JCxNEydzZbMY8L77hAH3aMokZzy2\nthis dataset has 1 validation errors\n", "", ""},
	}

	for _, c := range cases {
		dsr, err := f.DatasetRequests()
		if err != nil {
			t.Errorf("case \"%s\", error creating dataset request: %s", c.description, err)
			continue
		}

		opt := &SaveOptions{
			IOStreams:       streams,
			Ref:             c.ref,
			FilePath:        c.filepath,
			BodyPath:        c.bodypath,
			Title:           c.title,
			Message:         c.message,
			Publish:         c.publish,
			DryRun:          c.dryrun,
			DatasetRequests: dsr,
		}

		err = opt.Run()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case \"%s\", mismatched error. Expected: '%s', Got: '%v'", c.description, c.err, err)
			ioReset(in, out, errs)
			continue
		}

		if libErr, ok := err.(lib.Error); ok {
			if libErr.Message() != c.msg {
				t.Errorf("case \"%s\", mismatched user-friendly message. Expected: '%s', Got: '%s'", c.description, c.msg, libErr.Message())
				ioReset(in, out, errs)
				continue
			}
		} else if c.msg != "" {
			t.Errorf("case \"%s\", mismatched user-friendly message. Expected: '%s', Got: ''", c.description, c.msg)
			ioReset(in, out, errs)
			continue
		}

		if c.expect != out.String() {
			t.Errorf("case \"%s\", output mismatch. Expected: '%s', Got: '%s'", c.description, c.expect, out.String())
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}
}
