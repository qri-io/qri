package cmd

import (
	"context"
	"testing"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

func TestRenderComplete(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory()
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
		opt := &RenderOptions{
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if c.expect != opt.Refs.Ref() {
			t.Errorf("case %d, opt.Ref not set correctly. Expected: '%s', Got: '%s'", i, c.expect, opt.Refs.Ref())
			ioReset(in, out, errs)
			continue
		}

		if opt.RenderRequests == nil {
			t.Errorf("case %d, opt.RenderRequests not set.", i)
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}
}

func TestRenderRun(t *testing.T) {
	ctx := context.Background()
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	// set Default Template to something easier to work with, then
	// cleanup when test completes
	prevDefaultTemplate := base.DefaultTemplate
	base.DefaultTemplate = `<html><h1>{{ds.peername}}/{{ds.name}}</h1></html>`
	defer func() { base.DefaultTemplate = prevDefaultTemplate }()

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	templateFile := qfs.NewMemfileBytes("template.html", []byte(`<html><h2>{{ds.peername}}/{{ds.name}}</h2></html>`))

	if err := f.Init(); err != nil {
		t.Errorf("error initializing: %s", err)
		return
	}
	node, err := f.ConnectionNode()
	if err != nil {
		t.Errorf("error getting node from factory: %s", err)
		return
	}
	r := node.Repo

	key, err := r.Store().Put(ctx, templateFile)
	if err != nil {
		t.Errorf("error putting template into store: %s", err)
		return
	}

	cfg, err := f.Config()
	if err != nil {
		t.Errorf("error getting config from factory: %s", err)
		return
	}

	if err := cfg.Set("render.defaultTemplateHash", key); err != nil {
		t.Errorf("error setting default template in config: %s", err)
		return
	}

	cases := []struct {
		ref      string
		template string
		output   string
		expected string
		err      string
		msg      string
	}{
		{"", "", "", "", repo.ErrEmptyRef.Error(), "peername and dataset name needed in order to render, for example:\n   $ qri render me/dataset_name\nsee `qri render --help` from more info"},
		{"peer/bad_dataset", "", "", "", "unknown dataset 'peer/bad_dataset'", ""},
		{"peer/cities", "", "", "<html><h1>peer/cities</h1></html>", "", ""},
		{"peer/cities", "testdata/template.html", "", "<html><h2>peer/cities</h2><tbody><tr><td>toronto</td><td>40000000</td><td>55.5</td><td>false</td></tr><tr><td>new york</td><td>8500000</td><td>44.4</td><td>true</td></tr></tbody></html>", "", ""},
	}

	for i, c := range cases {
		rr, err := f.RenderRequests()
		if err != nil {
			t.Errorf("case %d, error creating dataset request: %s", i, err)
			continue
		}

		opt := &RenderOptions{
			IOStreams:      streams,
			Refs:           NewExplicitRefSelect(c.ref),
			Template:       c.template,
			Output:         c.output,
			RenderRequests: rr,
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
