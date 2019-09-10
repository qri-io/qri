// +build !windows

package fsi

import (
	"fmt"
	"path/filepath"
	"strings"
)

// setFileHidden ensures the filename begins with a dot. Other OSes may do more
func setFileHidden(path string) error {
	basename := filepath.Base(path)
	if !strings.HasPrefix(basename, ".") {
		return fmt.Errorf("hidden files must begin with \".\"")
	}
	return nil
}
