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
		{"diff two dataset metas",
			&DiffOptions{Left: "me/movies", Right: "me/cities", Selector: "meta"},
			"0 elements. 0 inserts. 0 deletes. 1 update.\n\n~ title: \"example city data\"\n",
			"", "",
		},
		{"diff json output",
			&DiffOptions{Left: "me/movies", Right: "me/cities", Selector: "meta", Format: "json"},
			`[{"type":"update","path":"/title","value":"example city data","originalValue":"example movie data"}]
`,
			"", "",
		},
	}

	for _, c := range cases {
		dsr, err := f.DatasetRequests()
		if err != nil {
			t.Errorf("case %s, error creating dataset request: %s", c.description, err)
			continue
		}

		opt := c.opt
		opt.IOStreams = streams
		opt.DatasetRequests = dsr

		err = opt.Run()
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %s, mismatched error. Expected: '%s', Got: '%v'", c.description, c.err, err)
			ioReset(in, out, errs)
			continue
		}

		if libErr, ok := err.(lib.Error); ok {
			if libErr.Message() != c.errMsg {
				t.Errorf("case %s, mismatched user-friendly message. Expected: '%s', Got: '%s'", c.description, c.errMsg, libErr.Message())
				ioReset(in, out, errs)
				continue
			}
		} else if c.errMsg != "" {
			t.Errorf("case %s, mismatched user-friendly message. Expected: '%s', Got: ''", c.description, c.errMsg)
			ioReset(in, out, errs)
			continue
		}

		if c.stdout != out.String() {
			t.Errorf("case %s, output mismatch. Expected: '%s', Got: '%s'", c.description, c.stdout, out.String())
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}
}
