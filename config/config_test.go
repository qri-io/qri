package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadFromFile(t *testing.T) {
	_, err := ReadFromFile("testdata/default.yaml")
	if err != nil {
		t.Errorf("error reading config: %s", err.Error())
		return
	}

	_, err = ReadFromFile("foobar")
	if err == nil {
		t.Error("expected read from bad path to error")
		return
	}
}

func TestWriteToFile(t *testing.T) {
	path := filepath.Join(os.TempDir(), "config.yaml")
	t.Log(path)
	cfg := DefaultConfig()
	if err := cfg.WriteToFile(path); err != nil {
		t.Errorf("error writing config: %s", err.Error())
		return
	}

	if err := cfg.WriteToFile("/not/a/path/foo"); err == nil {
		t.Error("expected write bad path to error")
		return
	}
}

func TestConfigSummaryString(t *testing.T) {
	summary := DefaultConfig().SummaryString()
	t.Log(summary)
	if !strings.Contains(summary, "API") {
		t.Errorf("expected summary to list API port")
	}
}

func TestConfigGet(t *testing.T) {
	cfg := DefaultConfig()
	cases := []struct {
		path   string
		expect interface{}
		err    string
	}{
		{"foo", nil, "invalid config path: foo"},
		{"p2p.enabled", true, ""},
		{"p2p.QriBootstrapAddrs.foo", nil, "invalid index value: foo"},
		{"p2p.QriBootstrapAddrs.0", cfg.P2P.QriBootstrapAddrs[0], ""},
		{"logging.levels.qriapi", cfg.Logging.Levels["qriapi"], ""},
		{"logging.levels.foo", nil, "invalid config path: logging.levels.foo"},
	}

	for i, c := range cases {
		got, err := cfg.Get(c.path)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}

		if !reflect.DeepEqual(c.expect, got) {
			t.Errorf("case %d result mismatch. expected: %v, got: %v", i, c.expect, got)
			continue
		}
	}
}

func TestConfigSet(t *testing.T) {
	cases := []struct {
		path  string
		value interface{}
		err   string
	}{
		{"foo", nil, "invalid config path: foo"},
		{"p2p.enabled", false, ""},
		{"p2p.qribootstrapaddrs.0", "wahoo", ""},
		{"p2p.qribootstrapaddrs.0", false, "invalid type for config path p2p.qribootstrapaddrs.0, expected: string, got: bool"},
	}

	for i, c := range cases {
		cfg := DefaultConfig()
		err := cfg.Set(c.path, c.value)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}

		if c.err != "" {
			continue
		}

		got, err := cfg.Get(c.path)
		if err != nil {
			t.Errorf("error getting set path: %s", err.Error())
			continue
		}

		if !reflect.DeepEqual(c.value, got) {
			t.Errorf("case %d result mismatch. expected: %v, got: %v", i, c.value, got)
			continue
		}
	}
}

func TestConfigValidate(t *testing.T) {
	if err := DefaultConfig().Validate(); err != nil {
		t.Errorf("error validating config: %s", err)
	}

	//  cases that should fail:
	// Profile:
	p := DefaultConfig()
	p.Profile.Type = "badType"
	if err := p.Validate(); err == nil {
		t.Error("When given bad input in Profile, config.Validate did not catch the error.")
	}
	// Repo:
	r := DefaultConfig()
	r.Repo.Type = "badType"
	if err := r.Validate(); err == nil {
		t.Error("When given bad input in Repo, config.Validate did not catch the error.")
	}

	// Store:
	s := DefaultConfig()
	s.Store.Type = "badType"
	if err := s.Validate(); err == nil {
		t.Error("When given bad input in Store, config.Validate did not catch the error.")
	}

	// Logging:
	l := DefaultConfig()
	l.Logging.Levels["qriapi"] = "badType"
	if err := l.Validate(); err == nil {
		t.Error("When given bad input in Logging, config.Validate did not catch the error.")
	}
}
