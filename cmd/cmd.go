package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Catch errors & pretty-print.
	// comment this out to get stack traces back.
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		if err, ok := r.(error); ok {
	// 			fmt.Println(err.Error())
	// 		} else {
	// 			fmt.Println(r)
	// 		}
	// 	}
	// }()

	if err := RootCmd.Execute(); err != nil {
		printErr(err)
		os.Exit(-1)
	}
}

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
