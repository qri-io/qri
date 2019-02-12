package lib

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/qri/config"
)

// func init() {
// 	Config = config.DefaultConfigForTesting()
// }

// func TestLoadConfig(t *testing.T) {
// 	path, err := ioutil.TempDir("", "config_tests")
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	defer os.RemoveAll(path)
// 	cfgPath := path + "/config.yaml"

// 	if err := config.DefaultConfigForTesting().WriteToFile(cfgPath); err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	if err := LoadConfig(ioes.NewDiscardIOStreams(), cfgPath); err != nil {
// 		t.Error(err.Error())
// 	}
// }

func TestConfigGet(t *testing.T) {
	cfgs := &Config{cfg: config.DefaultConfigForTesting()}
	p := &GetConfigParams{Field: "profile.id", Format: "json"}
	res := []byte{}
	if err := cfgs.Get(p, &res); err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(res, []byte(`"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"`)) {
		t.Errorf("response mismatch. got %s", string(res))
	}
}

// func TestSaveConfig(t *testing.T) {
// 	prevCFP := ConfigFilepath
// 	defer func() {
// 		ConfigFilepath = prevCFP
// 	}()

// 	path, err := ioutil.TempDir("", "save_config_test")
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	ConfigFilepath = ""
// 	if err := SaveConfig(); err == nil {
// 		t.Error("expected save to empty path to error")
// 	}

// 	ConfigFilepath = path + "/config.yaml"
// 	if err := SaveConfig(); err != nil {
// 		t.Error(err.Error())
// 	}
// }

func TestSetConfig(t *testing.T) {
	td, err := ioutil.TempDir("", "testSaveConfig")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(td)

	cfgs := &Config{
		cfg:      config.DefaultConfigForTesting(),
		filePath: filepath.Join(td, "config.yaml"),
	}

	if err := cfgs.Set(&config.Config{}); err == nil {
		t.Errorf("expected saving empty config to be invalid")
	}

	cfg := config.DefaultConfigForTesting()
	// SaveConfig = func() error { return fmt.Errorf("bad") }
	// if err := cfgs.Set(cfg); err == nil {
	// 	t.Errorf("expected saving error to return")
	// }

	cfg.Profile.Twitter = "@qri_io"
	if err := cfgs.Set(cfg); err != nil {
		t.Error(err.Error())
	}
	p := &GetConfigParams{Field: "profile.twitter", Format: "json"}
	res := []byte{}
	if err := cfgs.Get(p, &res); err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(res, []byte(`"@qri_io"`)) {
		t.Errorf("response mismatch. got %s", string(res))
	}
}
