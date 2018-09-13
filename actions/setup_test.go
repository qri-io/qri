package actions

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/qri-io/qri/config"
)

func TestSetupInitIPFSTeardown(t *testing.T) {
	path, err := ioutil.TempDir("", "SetupInitIPFSTeardown")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(path)

	cfg := config.DefaultConfigForTesting()
	cfg.Registry = nil
	if err := Setup(path, path+"/config.yaml", cfg); err != nil {
		t.Error(err.Error())
	}
	if err := InitIPFS(path, nil); err != nil {
		t.Error(err.Error())
	}
	if err := Teardown(path, config.DefaultConfigForTesting()); err != nil {
		t.Error(err.Error())
	}
}
