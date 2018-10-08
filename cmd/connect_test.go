package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/ioes"
	libtest "github.com/qri-io/qri/lib/test"
	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestConnect(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	_, registryServer := regmock.NewMockServer()

	path := filepath.Join(os.TempDir(), "qri_test_commands_connect")
	t.Logf("temp path: %s", path)

	//clean up if previous cleanup failed
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.RemoveAll(path)
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating test path: %s", err.Error())
		return
	}
	defer os.RemoveAll(path)

	cmd := "qri connect --setup --registry=" + registryServer.URL + " --disconnect-after=1"
	streams, _, _, _ := ioes.NewTestIOStreams()
	root := NewQriCommand(NewDirPathFactory(path), libtest.NewTestCrypto(), streams)

	defer func() {
		if e := recover(); e != nil {
			t.Errorf("unexpected panic:\n%s\n%s", cmd, e)
			return
		}
	}()

	_, err := executeCommand(root, cmd)
	if err != nil {
		t.Errorf("unexpected error executing command\n%s\n%s", cmd, err.Error())
		return
	}
}
