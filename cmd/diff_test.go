package cmd

import (
	"testing"
	"time"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
)

func TestDiffComplete(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(nil)
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
		opt := &DiffOptions{
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
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

func TestDiffRun(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	f, err := NewTestFactory(nil)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		description string
		opt         *DiffOptions
		stdout      string
		err         string
		errMsg      string
	}{
		{"diff with no options",
			&DiffOptions{},
			"",
			"repo: empty dataset reference",
			"",
		},
		{"diff with no options",
			&DiffOptions{Left: "me/movies", Right: "me/cities", Selector: "meta"},
			"0 elements. 0 inserts. 0 deletes. 1 update.\n\n~ title: \"example city data\"\n",
			"", "",
		},
		// {[]string{}, -1, "", "", ""},
		// {[]string{"me/bad_dataset"}, -1, "", "repo: not found", "could not find dataset 'me/bad_dataset'"},
		// {[]string{"me/movies"}, -1, "Diffd entire dataset 'peer/movies@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/QmcnKQTWNuAYpcqUa4Ss85WiNp4c9ySem4DNipxQzUqw5J'\n", "", ""},
		// {[]string{"me/cities", "me/counter"}, -1, "Diffd entire dataset 'peer/cities@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/QmSXfRRFy2T3gBARtvcf1GJAgg5dandb9fpqyUWAYzEcQq'\nDiffd entire dataset 'peer/counter@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/QmSULpGrhdSTVJh4C1jtnFsXNczMxXG6KaMLAhfSkM3zzu'\n", "", ""},
		// {[]string{"me/movies"}, -1, "", "repo: not found", "could not find dataset 'me/movies'"},
	}

	for i, c := range cases {
		dsr, err := f.DatasetRequests()
		if err != nil {
			t.Errorf("case %d, error creating dataset request: %s", i, err)
			continue
		}

		opt := c.opt
		opt.IOStreams = streams
		opt.DatasetRequests = dsr

		err = opt.Run()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
			ioReset(in, out, errs)
			continue
		}

		if libErr, ok := err.(lib.Error); ok {
			if libErr.Message() != c.errMsg {
				t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: '%s'", i, c.errMsg, libErr.Message())
				ioReset(in, out, errs)
				continue
			}
		} else if c.errMsg != "" {
			t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: ''", i, c.errMsg)
			ioReset(in, out, errs)
			continue
		}

		if c.stdout != out.String() {
			t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.stdout, out.String())
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}
}
