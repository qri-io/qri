package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
)

func TestUpdateComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_update_complete")
	defer run.Delete()

	f, err := NewTestFactory()
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
			IOStreams: run.Streams,
		}

		opt.Complete(f, c.args)

		if c.err != run.ErrStream.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, run.ErrStream.String())
			run.IOReset()
			continue
		}

		if c.ref != opt.Ref {
			t.Errorf("case %d, opt.From not set correctly. Expected: '%s', Got: '%s'", i, c.ref, opt.Ref)
			run.IOReset()
			continue
		}

		if opt.updateMethods == nil {
			t.Errorf("case %d, opt.updateMethods not set.", i)
			run.IOReset()
			continue
		}
		run.IOReset()
	}
}

func TestUpdateMethods(t *testing.T) {
	if err := confirmUpdateServiceNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_update_methods")
	defer run.Delete()

	tmpDir := run.MakeTmpDir(t, "update_methods")

	cfg := config.DefaultConfigForTesting().Copy()
	cfg.Update = &config.Update{Type: "mem"}
	cfg.Repo = &config.Repo{Type: "mem", Middleware: []string{}}
	cfg.Store = &config.Store{Type: "map"}

	inst, err := lib.NewInstance(context.Background(), tmpDir, lib.OptConfig(cfg), lib.OptIOStreams(run.Streams))
	if err != nil {
		t.Fatal(err)
	}

	o := &UpdateOptions{
		IOStreams:     run.Streams,
		Page:          1,
		PageSize:      25,
		inst:          inst,
		updateMethods: lib.NewUpdateMethods(inst),
	}

	run.IOReset()
	if err := o.Schedule([]string{"testdata/hello.sh", "R/PT1S"}); err != nil {
		t.Error(err)
	}

	expErrOutContains := "update scheduled, next update: 0001-01-01 00:00:01 +0000 UTC\n"
	if !strings.Contains(run.ErrStream.String(), expErrOutContains) {
		t.Errorf("schedule response mismatch. expected errOut:\n%s\ngot:\n%s", expErrOutContains, run.ErrStream.String())
	}

	run.IOReset()
	if err := o.List(); err != nil {
		t.Error(err)
	}

	// TODO (b5) - wee note on TestJobStringer, we should be testing times by setting local timezones
	listStdOutContains := "| shell"
	if !strings.Contains(run.OutStream.String(), listStdOutContains) {
		t.Errorf("list response mismatch. stdOut doesn't contain:\n%s\ngot:\n%s", listStdOutContains, run.OutStream.String())
	}

	// TODO (b5) - need to actually run an update here that generates a log entry
	// ideally by manually calling o.Run

	run.IOReset()
	if err := o.Logs([]string{}); err != nil {
		t.Error(err)
	}
	logsStdOut := "" // should be empty b/c this test runs no updates
	if logsStdOut != run.OutStream.String() {
		t.Errorf("logs response mismatch. expected errOut:\n%s\ngot:\n%s", logsStdOut, run.OutStream.String())
	}

	run.IOReset()
	if err := o.Unschedule([]string{"testdata/hello.sh"}); err != nil {
		t.Error(err)
	}
	unscheduleErrOutContains := "update unscheduled"
	if !strings.Contains(run.ErrStream.String(), unscheduleErrOutContains) {
		t.Errorf("logs response mismatch. errOut doesn't contain:\n%s\ngot:\n%s", unscheduleErrOutContains, run.ErrStream.String())
	}
}

func TestUpdateRun(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_update_run")
	defer run.Delete()

	f, err := NewTestFactory()
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
		run.IOReset()
		// dsr, err := f.DatasetRequests()
		// if err != nil {
		// 	t.Errorf("case %d, error creating dataset request: %s", i, err)
		// 	continue
		// }

		c.opt.IOStreams = run.Streams
		c.opt.updateMethods = lib.NewUpdateMethods(f.Instance())

		// TODO (b5): fix tests
		err = c.opt.RunUpdate(nil)
		if (err == nil && c.err != "") || (err != nil && c.err != err.Error()) {
			t.Errorf("case %d, mismatched error. Expected: '%s', Got: '%v'", i, c.err, err)
			continue
		}

		if libErr, ok := err.(errors.Error); ok {
			if libErr.Message() != c.msg {
				t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: '%s'", i, c.msg, libErr.Message())
				run.IOReset()
				continue
			}
		} else if c.msg != "" {
			t.Errorf("case %d, mismatched user-friendly message. Expected: '%s', Got: ''", i, c.msg)
			run.IOReset()
			continue
		}

		if c.expect != run.OutStream.String() {
			t.Errorf("case %d, output mismatch. Expected: '%s', Got: '%s'", i, c.expect, run.OutStream.String())
			continue
		}
	}
}
