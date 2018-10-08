package actions

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/qri-io/qri/config"
	libtest "github.com/qri-io/qri/lib/test"
)

func TestSetupInitIPFSTeardown(t *testing.T) {
	path, err := ioutil.TempDir("", "SetupInitIPFSTeardown")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(path)

	cfg := config.DefaultConfigForTesting()
	cfg.Registry = nil
	if err := Setup(path, path+"/config.yaml", cfg, true); err != nil {
		t.Error(err.Error())
	}
	g := libtest.NewTestCrypto()
	if err := g.GenerateEmptyIpfsRepo(path, ""); err != nil {
		t.Error(err.Error())
	}
	if err := Teardown(path, config.DefaultConfigForTesting()); err != nil {
		t.Error(err.Error())
	}
}
