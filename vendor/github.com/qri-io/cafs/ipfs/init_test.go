package ipfs_filestore

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestInitRepo(t *testing.T) {
	cases := []struct {
		configPath string
	}{
		{""},
		// {"./testdata/ipfs_test_config"},
	}

	for i, c := range cases {
		repoPath := filepath.Join(os.TempDir(), fmt.Sprintf("ipfs_init_test_repo_%d", i))
		if err := InitRepo(repoPath, c.configPath); err != nil {
			t.Error(err.Error())
		}
		if err := os.RemoveAll(repoPath); err != nil {
			t.Error(err.Error())
		}
	}
}
