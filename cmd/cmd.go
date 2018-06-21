package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

var log = golog.Logger("cmd")

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if os.Getenv("QRI_BACKTRACE") == "" {
		// Catch errors & pretty-print.
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					fmt.Println(err.Error())
				} else {
					fmt.Println(r)
				}
			}
		}()
	}

	root := NewQriCommand(EnvPathFactory, os.Stdin, os.Stdout, os.Stderr)
	if err := root.Execute(); err != nil {
		printErr(os.Stdin, err)
		os.Exit(-1)
	}
}

// ErrExit writes an error to the given io.Writer & exits
func ErrExit(w io.Writer, err error) {
	log.Debug(err.Error())
	if e, ok := err.(lib.Error); ok {
		if e.Message() != "" {
			printErr(w, fmt.Errorf(e.Message()))
			os.Exit(1)
		}
	}
	printErr(w, err)
	os.Exit(1)
}

// ExitIfErr only calls ErrExit if there is an error present
func ExitIfErr(w io.Writer, err error) {
	if err != nil {
		ErrExit(w, err)
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

// parseSecrets turns a key,value sequence into a map[string]string
func parseSecrets(secrets ...string) (map[string]string, error) {
	if len(secrets)%2 != 0 {
		return nil, fmt.Errorf("expected even number of (key,value) pairs for secrets")
	}
	s := map[string]string{}
	for i := 0; i < len(secrets); i = i + 2 {
		s[secrets[i]] = secrets[i+1]
	}
	return s, nil
}

// parseCmdLineDatasetRef parses DatasetRefs, assuming peer "me" if none given.
func parseCmdLineDatasetRef(ref string) (repo.DatasetRef, error) {
	if !strings.ContainsAny(ref, "@/") {
		ref = "me/" + ref
	}
	return repo.ParseDatasetRef(ref)
}
