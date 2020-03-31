package cmd

import (
	"testing"
)

func TestGetComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_get_complete")
	defer run.Delete()

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args     []string
		selector string
		refs     []string
		err      string
	}{
		{[]string{}, "", []string{}, ""},
		{[]string{"one arg"}, "", []string{"one arg"}, ""},
		{[]string{"commit", "peer/ds"}, "commit", []string{"peer/ds"}, ""},
		{[]string{"commit.author", "peer/ds"}, "commit.author", []string{"peer/ds"}, ""},
		// TODO(dlong): Fix tests when `qri get` can be passed multiple arguments.
		//{[]string{"peer/ds_two", "peer/ds"}, "", []string{"peer/ds_two", "peer/ds"}, ""},
		//{[]string{"foo", "peer/ds"}, "", []string{"foo", "peer/ds"}, ""},
		{[]string{"structure"}, "structure", []string{}, ""},
		{[]string{"stats", "me/cities"}, "stats", []string{"me/cities"}, ""},
		{[]string{"stats", "me/sitemap"}, "stats", []string{"me/sitemap"}, ""},
	}

	for i, c := range cases {
		opt := &GetOptions{
			IOStreams: run.Streams,
		}

		opt.Complete(f, c.args)

		if c.err != run.ErrStream.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, run.ErrStream.String())
			run.IOReset()
			continue
		}

		if !testSliceEqual(c.refs, opt.Refs.RefList()) {
			t.Errorf("case %d, opt.Refs not set correctly. Expected: '%q', Got: '%q'", i, c.refs, opt.Refs.RefList())
			run.IOReset()
			continue
		}

		if c.selector != opt.Selector {
			t.Errorf("case %d, opt.Selector not set correctly. Expected: '%s', Got: '%s'", i, c.selector, opt.Selector)
			run.IOReset()
			continue
		}

		if opt.DatasetRequests == nil {
			t.Errorf("case %d, opt.DatasetRequests not set.", i)
			run.IOReset()
			continue
		}
		run.IOReset()
	}
}
