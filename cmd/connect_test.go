package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestConnect(t *testing.T) {

	_, registryServer := regmock.NewMockServer()

	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

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

	// defer func() {
	// 	if e := recover(); e != nil {
	// 		t.Errorf("unexpected panic:\n%s\n%s", strings.Join(args, " "), e)
	// 		return
	// 	}
	// }()

  args := []string{"connect", "--setup", "--registry=" + registryServer.URL, "--disconnect-after=3"}
	_, in, out, errs := NewTestIOStreams()
	root := NewQriCommand(NewDirPathFactory(path), in, out, errs)

	_, err := executeCommand(root, args...)
	if err != nil {
		t.Errorf("unexpected error executing command\n%s\n%s", strings.Join(args, " "), err.Error())
		return
	}
}
