package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConnect(t *testing.T) {

	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	path := filepath.Join(os.TempDir(), "qri_test_commands_connect")
	t.Logf("temp path: %s", path)
	os.Setenv("IPFS_PATH", filepath.Join(path, "ipfs"))
	os.Setenv("QRI_PATH", filepath.Join(path, "qri"))

	//clean up if previous cleanup failed
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.RemoveAll(path)
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating test path: %s", err.Error())
		return
	}
	defer os.RemoveAll(path)

	args := []string{"connect", "--setup", "--disconnect-after=3"}

	// defer func() {
	// 	if e := recover(); e != nil {
	// 		t.Errorf("unexpected panic:\n%s\n%s", strings.Join(args, " "), e)
	// 		return
	// 	}
	// }()

	_, err := executeCommand(RootCmd, args...)
	if err != nil {
		t.Errorf("unexpected error executing command\n%s\n%s", strings.Join(args, " "), err.Error())
		return
	}
}
