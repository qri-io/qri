package lib

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/qri-io/qri/config"
)

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

func TestGetConfig(t *testing.T) {
	cfg := config.DefaultConfigForTesting()
	// TODO (b5) - hack until we can get better test-instance allocation
	inst := NewInstanceFromConfigAndNode(cfg, nil)
	m := NewConfigMethods(inst)

	p := &GetConfigParams{Field: "profile.id", Format: "json"}
	res := []byte{}
	if err := m.GetConfig(p, &res); err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(res, []byte(`"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"`)) {
		t.Errorf("response mismatch. got %s", string(res))
	}
}

func TestSaveConfigToFile(t *testing.T) {
	path, err := ioutil.TempDir("", "save_config_test")
	if err != nil {
		t.Fatal(err.Error())
	}

	cfgPath := path + "/config.yaml"
	cfg := config.DefaultConfigForTesting()
	cfg.SetPath(cfgPath)
	// TODO (b5) - hack until we can get better test-instance allocation
	inst := NewInstanceFromConfigAndNode(cfg, nil)
	m := NewConfigMethods(inst)

	var ok bool
	if err := m.SetConfig(cfg, &ok); err != nil {
		t.Error(err.Error())
	}
}

func TestSetConfig(t *testing.T) {
	cfg := config.DefaultConfigForTesting()
	// TODO (b5) - hack until we can get better test-instance allocation
	inst := NewInstanceFromConfigAndNode(cfg, nil)
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
	res := []byte{}
	if err := m.GetConfig(p, &res); err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal(res, []byte(`"@qri_io"`)) {
		t.Errorf("response mismatch. got %s", string(res))
	}
}
