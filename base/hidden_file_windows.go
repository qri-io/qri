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

// WriteHiddenFile writes content to the file and sets the hidden attribute on it. For a hidden
// file, using ioutil.WriteFile instead will result in "Access is denied." so use this instead.
// Windows specific functionality
func WriteHiddenFile(path, content string) error {
	// Ensure the filename begins with a dot
	basename := filepath.Base(path)
	if !strings.HasPrefix(basename, ".") {
		return fmt.Errorf("hidden files must begin with \".\"")
	}

	// Create the file to write, making sure to set the hidden file attribute
	filenamePtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	h, err := syscall.CreateFile(
		filenamePtr, // filename
		syscall.GENERIC_WRITE, // access
		uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE), // share mode
		nil, // security attributes
		syscall.CREATE_ALWAYS, // creation disposition
		syscall.FILE_ATTRIBUTE_HIDDEN, // flags and attributes
		0, // template file handle
	)
	if err != nil {
		return err
	}
	defer syscall.Close(h)

	// Write contents
	_, err = syscall.Write(h, []byte(content))
	if err != nil {
		return err
	}
	return nil
}
