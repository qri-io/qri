package cmd

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
)

func TestUpdateComplete(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(nil)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args []string
		ref  string
		err  string
	}{
		{[]string{}, "", ""},
		{[]string{"me/from"}, "me/from", ""},
		{[]string{"me/from", "me/to"}, "", ""},
	}

	for i, c := range cases {
		opt := &UpdateOptions{
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if c.ref != opt.Ref {
			t.Errorf("case %d, opt.From not set correctly. Expected: '%s', Got: '%s'", i, c.ref, opt.Ref)
			ioReset(in, out, errs)
			continue
		}

		if opt.updateMethods == nil {
			t.Errorf("case %d, opt.updateMethods not set.", i)
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}
}

func TestUpdateMethods(t *testing.T) {
	if err := confirmUpdateServiceNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	tmpDir, err := ioutil.TempDir("", "update_methods")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	cfg := config.DefaultConfigForTesting().Copy()
	cfg.Update = &config.Update{Type: "mem"}
	cfg.Repo = &config.Repo{Type: "mem", Middleware: []string{}}
	cfg.Store = &config.Store{Type: "map"}

	inst, err := lib.NewInstance(tmpDir, lib.OptConfig(cfg), lib.OptIOStreams(streams))
	if err != nil {
		t.Fatal(err)
	}

	o := &UpdateOptions{
		IOStreams:     streams,
		Page:          1,
		PageSize:      25,
		inst:          inst,
		updateMethods: lib.NewUpdateMethods(inst),
	}

	ioReset(in, out, errs)
	if err := o.Schedule([]string{"testdata/hello.sh", "R/PT1S"}); err != nil {
		t.Error(err)
	}

	expErrOutContains := "update scheduled, next update: 0001-01-01 00:00:01 +0000 UTC\n"
	if !strings.Contains(errs.String(), expErrOutContains) {
		t.Errorf("schedule response mismatch. expected errOut:\n%s\ngot:\n%s", expErrOutContains, errs.String())
	}

	ioReset(in, out, errs)
	if err := o.List(); err != nil {
		t.Error(err)
	}
	listStdOutContains := "shell | 0001-01-01 00:00:01 +0000 UTC"
	if !strings.Contains(out.String(), listStdOutContains) {
		t.Errorf("list response mismatch. stdOut doesn't contain:\n%s\ngot:\n%s", listStdOutContains, out.String())
	}

	// TODO (b5) - need to actually run an update here that generates a log entry
	// ideally by manually calling o.Run

	ioReset(in, out, errs)
	if err := o.Logs([]string{}); err != nil {
		t.Error(err)
	}
	logsStdOut := "" // should be empty b/c this test runs no updates
	if logsStdOut != out.String() {
		t.Errorf("logs response mismatch. expected errOut:\n%s\ngot:\n%s", logsStdOut, out.String())
	}

	ioReset(in, out, errs)
	if err := o.Unschedule([]string{"testdata/hello.sh"}); err != nil {
		t.Error(err)
	}
	unscheduleErrOutContains := "update unscheduled"
	if !strings.Contains(errs.String(), unscheduleErrOutContains) {
		t.Errorf("logs response mismatch. errOut doesn't contain:\n%s\ngot:\n%s", unscheduleErrOutContains, errs.String())
	}
}

func TestUpdateRun(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(nil)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		opt              *UpdateOptions
		expect, msg, err string
	}{
		// TODO (b5): fix tests
		// {&UpdateOptions{Ref: "other/bad_ref"}, "", "", "unknown dataset 'other/bad_ref'. please add before updating"},
		// {&UpdateOptions{Ref: "me/bad_ref"}, "", "", "unknown dataset 'peer/bad_ref'. please add before updating"},
	}

	for i, c := range cases {
		ioReset(in, out, errs)
		// dsr, err := f.DatasetRequests()
		// if err != nil {
		// 	t.Errorf("case %d, error creating dataset request: %s", i, err)
		// 	continue
		// }

		c.opt.IOStreams = streams
		c.opt.updateMethods = lib.NewUpdateMethods(f.Instance())

		// TODO (b5): fix tests
		err = c.opt.RunUpdate(nil)
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
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

		if c.expect != out.String() {
			t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.expect, out.String())
			continue
		}
	}
}
