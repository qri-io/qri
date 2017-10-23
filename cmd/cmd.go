package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/viper"
)

// ErrExit writes an error to stdout & exits
func ErrExit(err error) {
	PrintErr(err)
	os.Exit(1)
}

func ExitIfErr(err error) {
	if err != nil {
		// PrintErr(err)
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
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
