package cmd

import (
	"runtime"
	"testing"
)

func TestDoesCommandExist(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if doesCommandExist("ls") == false {
			t.Error("ls command does not exist!")
		}
		if doesCommandExist("ls111") == true {
			t.Error("ls111 command should not exist!")
		}
	}
}
