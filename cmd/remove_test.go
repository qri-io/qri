package cmd

import (
	"testing"
	"time"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/lib"
)

func TestRemoveComplete(t *testing.T) {
	streams, in, out, errs := NewTestIOStreams()
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
		opt := &RemoveOptions{
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if !testSliceEqual(c.args, opt.Args) {
			t.Errorf("case %d, opt.Args not set correctly. Expected: '%s', Got: '%s'", i, c.args, opt.Args)
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

func TestRemoveValidate(t *testing.T) {
	cases := []struct {
		args []string
		err  string
		msg  string
	}{
		{[]string{}, lib.ErrBadArgs.Error(), "please specify a dataset path or name you would like to remove from your qri node"},
		{[]string{"me/test"}, "", ""},
		{[]string{"me/test", "me/test2"}, "", ""},
	}
	for i, c := range cases {
		opt := &RemoveOptions{
			Args: c.args,
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

func TestRemoveRun(t *testing.T) {
	streams, in, out, errs := NewTestIOStreams()
	setNoColor(true)

	// in order to have consistent responses
	// we need to artificially specify the timestamp
	// we use the dsfs.Timestamp func variable to override
	// the actual time
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	f, err := NewTestFactory(nil)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args     []string
		expected string
		err      string
		msg      string
	}{
		{[]string{}, "", "", ""},
		{[]string{"me/bad_dataset"}, "", "repo: not found", "could not find dataset 'peer/bad_dataset'"},
		{[]string{"me/movies"}, "removed dataset 'peer/movies@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/QmQ7WC69LZh8TgzT9nWzCrmKQeFLcYiunT6viZEgNFwaYV'\n", "", ""},
		{[]string{"me/cities", "me/counter"}, "removed dataset 'peer/cities@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/QmSgwXYRCYfyjpKrm7AvajcNEeh3ppCAPKNhHxFM4hdNAA'\nremoved dataset 'peer/counter@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/map/Qmbh9W1KVq9kdPWZqVELrrRNiA2AwskRic15jK3vafizLA'\n", "", ""},
		{[]string{"me/movies"}, "", "repo: not found", "could not find dataset 'peer/movies'"},
	}

	for i, c := range cases {
		dsr, err := f.DatasetRequests()
		if err != nil {
			t.Errorf("case %d, error creating dataset request: %s", i, err)
			continue
		}

		opt := &RemoveOptions{
			IOStreams:       streams,
			Args:            c.args,
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

func testSliceEqual(a, b []string) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
