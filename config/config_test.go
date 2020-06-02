package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
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
	cfg := DefaultConfigForTesting()
	if err := cfg.WriteToFile(path); err != nil {
		t.Errorf("error writing config: %s", err.Error())
		return
	}

	if err := cfg.WriteToFile("/not/a/path/foo"); err == nil {
		t.Error("expected write bad path to error")
		return
	}
}

func TestWriteToFileWithExtraData(t *testing.T) {
	path := filepath.Join(os.TempDir(), "config.yaml")
	t.Log(path)
	cfg := Config{
		Revision: CurrentConfigRevision,
		Profile: &ProfilePod{
			ID:       "QmU27VdAEUL5NGM6oB56htTxvHLfcGZgsgxrJTdVr2k4zs",
			Peername: "test_peername",
			Created:  time.Unix(1234567890, 0).In(time.UTC),
			Updated:  time.Unix(1234567890, 0).In(time.UTC),
		},
	}
	cfg.Profile.PeerIDs = []string{"/test_network/testPeerID"}
	cfg.Profile.NetworkAddrs = []string{"foo", "bar", "baz"}

	if err := cfg.WriteToFile(path); err != nil {
		t.Errorf("error writing config: %s", err.Error())
		return
	}

	golden := "testdata/simple.yaml"
	f1, err := ioutil.ReadFile(golden)
	if err != nil {
		t.Errorf("error reading golden file: %s", err.Error())
	}
	f2, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("error reading written file: %s", err.Error())
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(f1), string(f2), false)
	if len(diffs) > 1 {
		fmt.Println(dmp.DiffPrettyText(diffs))
		t.Errorf("failed to match: %s <> %s", golden, path)
	}
}

func TestConfigSummaryString(t *testing.T) {
	summary := DefaultConfigForTesting().SummaryString()
	t.Log(summary)
	if !strings.Contains(summary, "API") {
		t.Errorf("expected summary to list API port")
	}
}

func TestConfigGet(t *testing.T) {
	cfg := DefaultConfigForTesting()
	cases := []struct {
		path   string
		expect interface{}
		err    string
	}{
		{"foo", nil, "at \"foo\": path not found"},
		{"p2p.enabled", true, ""},
		{"p2p.QriBootstrapAddrs.foo", nil, "at \"p2p.QriBootstrapAddrs.foo\": need int, got string: \"foo\""},
		{"p2p.QriBootstrapAddrs.0", cfg.P2P.QriBootstrapAddrs[0], ""},
		{"logging.levels.qriapi", cfg.Logging.Levels["qriapi"], ""},
		{"logging.levels.foo", nil, "at \"logging.levels.foo\": invalid path"},
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
		{"foo", nil, "at \"foo\": path not found"},
		{"p2p.enabled", false, ""},
		{"p2p.qribootstrapaddrs.0", "wahoo", ""},
		{"p2p.qribootstrapaddrs.0", false, "at \"p2p.qribootstrapaddrs.0\": need string, got bool: false"},
		{"logging.levels.qriapi", "debug", ""},
	}

	for i, c := range cases {
		cfg := DefaultConfigForTesting()

		err := cfg.Set(c.path, c.value)
		if err != nil {
			if err.Error() != c.err {
				t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			}
			continue
		} else if c.err != "" {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
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

func TestImmutablePaths(t *testing.T) {
	dc := DefaultConfigForTesting()
	for path := range ImmutablePaths() {
		if _, err := dc.Get(path); err != nil {
			t.Errorf("path %s default configuration error: %s", path, err.Error())
		}
	}
}

func TestConfigValidate(t *testing.T) {
	if err := DefaultConfigForTesting().Validate(); err != nil {
		t.Errorf("error validating config: %s", err)
	}

	//  cases that should fail:
	p := DefaultConfigForTesting()

	// Profile:
	p.Profile = nil
	if err := p.Validate(); err == nil {
		t.Error("When given no Profile, config.Validate did not catch the error.")
	}
	p.Profile = DefaultProfileForTesting()
	p.Profile.Type = "badType"
	if err := p.Validate(); err == nil {
		t.Error("When given bad input in Profile, config.Validate did not catch the error.")
	}
	// Repo:
	r := DefaultConfigForTesting()
	r.Repo.Type = "badType"
	if err := r.Validate(); err == nil {
		t.Error("When given bad input in Repo, config.Validate did not catch the error.")
	}

	// Store:
	s := DefaultConfigForTesting()
	s.Store.Type = "badType"
	if err := s.Validate(); err == nil {
		t.Error("When given bad input in Store, config.Validate did not catch the error.")
	}

	// Logging:
	l := DefaultConfigForTesting()
	l.Logging.Levels["qriapi"] = "badType"
	if err := l.Validate(); err == nil {
		t.Error("When given bad input in Logging, config.Validate did not catch the error.")
	}
}

func TestConfigCopy(t *testing.T) {
	cases := []struct {
		config *Config
	}{
		{DefaultConfigForTesting()},
	}
	for i, c := range cases {
		cpy := c.config.Copy()
		if !reflect.DeepEqual(cpy, c.config) {
			t.Errorf("Config Copy test case %v, config structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.config)
			continue
		}
		cpy.API.AllowedOrigins[0] = ""
		if reflect.DeepEqual(cpy, c.config) {
			t.Errorf("Config Copy test case %v, editing one config struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.config)
			continue
		}
	}
}

func TestConfigIgnoreFields(t *testing.T) {
	if _, err := ReadFromFile("testdata/deprecated.yaml"); err != nil {
		t.Error(err)
	}
}

func TestConfigInvalidFields(t *testing.T) {
	if _, err := ReadFromFile("testdata/invalid.yaml"); err == nil {
		t.Errorf("expected error, got none")
	}
}
