package cmd

import (
	"testing"
)

func TestValidateComplete(t *testing.T) {
	// in, out, and errs are buffers
	streams, in, out, errs := NewTestIOStreams()

	f, err := NewTestFactory(streams)
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
		{[]string{"foo", "bar"}, "foo", ""},
	}

	for i, c := range cases {
		opt := &ValidateOptions{
			IOStreams: f.IOStreams,
		}
		opt.Complete(f, c.args)

		if c.err != "" && errs.String() != c.err {
			t.Errorf("case %v, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			continue
		}

		if errs.Len() != 0 {
			t.Errorf("case %v, unexpected error: %s", i, errs.String())
			continue
		}

		if opt.Ref != c.expect {
			t.Errorf("case %v, opt.Ref not set correctly. Expected: '%s', Got: '%s'", i, c.expect, opt.Ref)
			continue
		}
		in.Reset()
		out.Reset()
		errs.Reset()
	}

}
