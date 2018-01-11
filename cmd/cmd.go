package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// ErrExit writes an error to stdout & exits
func ErrExit(err error) {
	printErr(err)
	os.Exit(1)
}

// ExitIfErr panics if an error is present
func ExitIfErr(err error) {
	if err != nil {
		// printErr(err)
		panic(err)
		os.Exit(1)
	}
}

// GetWd is a convenience method to get the working
// directory or bail.
func GetWd() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %s", err.Error())
		os.Exit(1)
	}

	return dir
}

// cachePath returns the configurable place to keep data
func cachePath() string {
	return viper.GetString("cache")
}

func userHomeDir() string {
	dir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return dir

	// if runtime.GOOS == "windows" {
	// 	home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	// 	if home == "" {
	// 		home = os.Getenv("USERPROFILE")
	// 	}
	// 	return home
	// }
	// return os.Getenv("HOME")
}

func loadFileIfPath(path string) (file *os.File, err error) {
	if path == "" {
		return nil, nil
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	return os.Open(path)
}
