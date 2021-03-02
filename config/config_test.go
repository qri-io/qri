package config_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestReadFromFile(t *testing.T) {
	_, err := config.ReadFromFile("testdata/default.yaml")
	if err != nil {
		t.Errorf("error reading config: %s", err.Error())
		return
	}

	_, err = config.ReadFromFile("foobar")
	if err == nil {
		t.Error("expected read from bad path to error")
		return
	}
}

func TestWriteToFile(t *testing.T) {
	path := filepath.Join(os.TempDir(), "config.yaml")
	t.Log(path)
	cfg := testcfg.DefaultConfigForTesting()
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
	cfg := config.Config{
		Revision: config.CurrentConfigRevision,
		Profile: &config.ProfilePod{
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
	summary := testcfg.DefaultConfigForTesting().SummaryString()
	t.Log(summary)
	if !strings.Contains(summary, "API") {
		t.Errorf("expected summary to list API port")
	}
}

func TestConfigGet(t *testing.T) {
	cfg := testcfg.DefaultConfigForTesting()
	cases := []struct {
		path   string
		expect interface{}
		err    string
	}{
		{"foo", nil, "at \"foo\": not found"},
		{"p2p.enabled", true, ""},
		{"p2p.QriBootstrapAddrs.foo", nil, "at \"p2p.QriBootstrapAddrs.foo\": need int, got string: \"foo\""},
		{"p2p.QriBootstrapAddrs.0", cfg.P2P.QriBootstrapAddrs[0], ""},
		{"logging.levels.qriapi", cfg.Logging.Levels["qriapi"], ""},
		{"logging.levels.foo", nil, "at \"logging.levels.foo\": not found"},
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
		{"foo", nil, "at \"foo\": not found"},
		{"p2p.enabled", false, ""},
		{"p2p.qribootstrapaddrs.0", "wahoo", ""},
		{"p2p.qribootstrapaddrs.0", false, "at \"p2p.qribootstrapaddrs.0\": need string, got bool: false"},
		{"logging.levels.qriapi", "debug", ""},
	}

	for i, c := range cases {
		cfg := testcfg.DefaultConfigForTesting()

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
	dc := testcfg.DefaultConfigForTesting()
	for path := range config.ImmutablePaths() {
		if _, err := dc.Get(path); err != nil {
			t.Errorf("path %s default configuration error: %s", path, err.Error())
		}
	}
}

func TestConfigValidate(t *testing.T) {
	if err := testcfg.DefaultConfigForTesting().Validate(); err != nil {
		t.Errorf("error validating config: %s", err)
	}

	//  cases that should fail:
	p := testcfg.DefaultConfigForTesting()

	// Profile:
	p.Profile = nil
	if err := p.Validate(); err == nil {
		t.Error("When given no Profile, config.Validate did not catch the error.")
	}
	p.Profile = testcfg.DefaultProfileForTesting()
	p.Profile.Type = "badType"
	if err := p.Validate(); err == nil {
		t.Error("When given bad input in Profile, config.Validate did not catch the error.")
	}
	// Repo:
	r := testcfg.DefaultConfigForTesting()
	r.Repo.Type = "badType"
	if err := r.Validate(); err == nil {
		t.Error("When given bad input in Repo, config.Validate did not catch the error.")
	}

	// Logging:
	l := testcfg.DefaultConfigForTesting()
	l.Logging.Levels["qriapi"] = "badType"
	if err := l.Validate(); err == nil {
		t.Error("When given bad input in Logging, config.Validate did not catch the error.")
	}
}

func TestConfigCopy(t *testing.T) {
	cases := []struct {
		config *config.Config
	}{
		{testcfg.DefaultConfigForTesting()},
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
