package base

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
)

// SetFileHidden sets the hidden attribute on the given file. Windows specific functionality
func SetFileHidden(path string) error {
	basename := filepath.Base(path)
	if !strings.HasPrefix(basename, ".") {
		return fmt.Errorf("hidden files must begin with \".\"")
	}
	filenamePtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return syscall.SetFileAttributes(filenamePtr, syscall.FILE_ATTRIBUTE_HIDDEN)
}
