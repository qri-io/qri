// Package cmd defines the CLI interface. It relies heavily on the spf13/cobra
// package. Much of its structure is adapted from kubernetes/kubernetes/tree/master/cmd
// The `help` message for each command uses backticks rather than quotes when
// refering to commands by name, even though it is cumbersome to maintain.
// Using backticks means we can get better formatting when auto generating markdown
// documentation from the command help messages.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/gen"
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

	ensureLargeNumOpenFiles()

	// root context
	ctx := context.Background()

	root := NewQriCommand(ctx, EnvPathFactory, gen.NewCryptoSource(), ioes.NewStdIOStreams())
	// If the subcommand hits an error, don't show usage or the error, since we'll show
	// the error message below, on our own. Usage is still shown if the subcommand
	// is missing command-line arguments.
	root.SilenceUsage = true
	root.SilenceErrors = true
	// Execute the subcommand
	if err := root.Execute(); err != nil {
		printErr(os.Stderr, err)
		os.Exit(1)
	}
}

// ErrExit writes an error to the given io.Writer & exits
func ErrExit(w io.Writer, err error) {
	log.Debug(err.Error())
	if e, ok := err.(lib.Error); ok && e.Message() != "" {
		printErr(w, fmt.Errorf(e.Message()))
	} else {
		printErr(w, err)
	}
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

// currentPath is used for test purposes to get the path from which qri is executing
func currentPath() (string, bool) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", ok
	}
	return path.Dir(filename), true
}
