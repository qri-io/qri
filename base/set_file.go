// +build !windows

package base

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SetFileHidden ensures the filename begins with a dot. Other OSes may do more
func SetFileHidden(path string) error {
	basename := filepath.Base(path)
	if !strings.HasPrefix(basename, ".") {
		return fmt.Errorf("hidden files must begin with \".\"")
	}
	return nil
}
