package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
)

func TestEnvPathFactory(t *testing.T) {
	//Needed to clean up changes after the test has finished running
	prevQRIPath := os.Getenv("QRI_PATH")
	prevIPFSPath := os.Getenv("IPFS_PATH")

	defer func() {
		os.Setenv("QRI_PATH", prevQRIPath)
		os.Setenv("IPFS_PATH", prevIPFSPath)
	}()

	//Test variables
	emptyPath := ""
	fakePath := "fake_path"
	home, err := homedir.Dir()
	if err != nil {
		t.Fatalf("Failed to find the home directory: %s", err.Error())
	}

	tests := []struct {
		qriPath    string
		ipfsPath   string
		qriAnswer  string
		ipfsAnswer string
	}{
		{emptyPath, emptyPath, filepath.Join(home, ".qri"), filepath.Join(home, ".ipfs")},
		{emptyPath, fakePath, filepath.Join(home, ".qri"), fakePath},
		{fakePath, emptyPath, fakePath, filepath.Join(home, ".ipfs")},
		{fakePath, fakePath, fakePath, fakePath},
	}

	for i, test := range tests {
		err := os.Setenv("QRI_PATH", test.qriPath)
		if err != nil {
			t.Errorf("case %d failed to set up QRI_PATH: %s", i, err.Error())
		}

		err = os.Setenv("IPFS_PATH", test.ipfsPath)
		if err != nil {
			t.Errorf("case %d failed to set up IPFS_PATH: %s", i, err.Error())
		}

		qriResult, ipfsResult := EnvPathFactory()

		if !strings.EqualFold(qriResult, test.qriAnswer) {
			t.Errorf("case %d expected qri path to be %s, but got %s", i, test.qriAnswer, qriResult)
		}

		if !strings.EqualFold(ipfsResult, test.ipfsAnswer) {
			t.Errorf("case %d Expected ipfs path to be %s, but got %s", i, test.ipfsAnswer, ipfsResult)
		}

	}
}
