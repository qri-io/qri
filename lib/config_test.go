package lib

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/p2p"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestGetConfig(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cfg := testcfg.DefaultConfigForTesting()
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err)
		return
	}
	node, err := p2p.NewQriNode(mr, cfg.P2P, event.NilBus, nil)
	if err != nil {
		t.Fatal(err)
	}

	inst := NewInstanceFromConfigAndNode(ctx, cfg, node)
	m := NewConfigMethods(inst)

	p := &GetConfigParams{Field: "profile.id", Format: "json"}
	res, err := m.GetConfig(ctx, p)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(res, []byte(`"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"`)) {
		t.Errorf("response mismatch. got %s", string(res))
	}
}

func TestSaveConfigToFile(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	path, err := ioutil.TempDir("", "save_config_test")
	if err != nil {
		t.Fatal(err.Error())
	}

	cfgPath := path + "/config.yaml"
	cfg := testcfg.DefaultConfigForTesting()
	cfg.SetPath(cfgPath)
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err)
		return
	}
	node, err := p2p.NewQriNode(mr, cfg.P2P, event.NilBus, nil)
	if err != nil {
		t.Fatal(err)
	}

	inst := NewInstanceFromConfigAndNode(ctx, cfg, node)
	m := NewConfigMethods(inst)

	var ok bool
	if err := m.SetConfig(cfg, &ok); err != nil {
		t.Error(err.Error())
	}
}

func TestSetConfig(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	cfg := testcfg.DefaultConfigForTesting()
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err)
		return
	}
	node, err := p2p.NewQriNode(mr, cfg.P2P, event.NilBus, nil)
	if err != nil {
		t.Fatal(err)
	}

	inst := NewInstanceFromConfigAndNode(ctx, cfg, node)
	m := NewConfigMethods(inst)

	var set bool

	if err := m.SetConfig(&config.Config{}, &set); err == nil {
		t.Errorf("expected saving empty config to be invalid")
	}

	cfg.Profile.Twitter = "@qri_io"
	if err := m.SetConfig(cfg, &set); err != nil {
		t.Error(err.Error())
	}
	p := &GetConfigParams{Field: "profile.twitter", Format: "json"}
	res, err := m.GetConfig(ctx, p)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(res, []byte(`"@qri_io"`)) {
		t.Errorf("response mismatch. got %s", string(res))
	}
}
