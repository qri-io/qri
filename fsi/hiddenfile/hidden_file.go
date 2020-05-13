// +build !windows

package hiddenfile

import (
	"fmt"
	"io/ioutil"
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

// WriteHiddenFile ensures the filename begins with a dot, and writes content to it. Other
// Operating Systems may do more
func WriteHiddenFile(path, content string) error {
	// Ensure the filename begins with a dot
	basename := filepath.Base(path)
	if !strings.HasPrefix(basename, ".") {
		return fmt.Errorf("hidden files must begin with \".\"")
	}
	// Write the contents
	if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	return nil
}
