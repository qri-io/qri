// Package cmd defines the CLI interface. It relies heavily on the spf13/cobra
// package. Much of its structure is adapted from kubernetes/kubernetes/tree/master/cmd
// The `help` message for each command uses backticks rather than quotes when
// referring to commands by name, even though it is cumbersome to maintain.
// Using backticks means we can get better formatting when auto generating markdown
// documentation from the command help messages.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs/qipfs"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/config/migrate"
	qrierr "github.com/qri-io/qri/errors"
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
	ctors := Constructors{
		CryptoGenerator: key.NewCryptoGenerator(),
		InitIPFS:        qipfs.InitRepo,
	}
	root, shutdown := NewQriCommand(ctx, StandardRepoPath(), ctors, ioes.NewStdIOStreams())
	// If the subcommand hits an error, don't show usage or the error, since we'll show
	// the error message below, on our own. Usage is still shown if the subcommand
	// is missing command-line arguments.
	root.SilenceUsage = true
	root.SilenceErrors = true
	// Execute the subcommand
	if err := root.Execute(); err != nil {
		ErrExit(os.Stderr, err)
	}

	<-shutdown()
}

const (
	// ExitCodeOK is a 0 exit code. we're good! success! yay!
	ExitCodeOK = iota
	// ExitCodeErr is a generic error exit code, all non-special errors occur here
	ExitCodeErr
	// ExitCodeNeedMigration indicates a required migration
	ExitCodeNeedMigration
)

// ErrExit writes an error to the given io.Writer & exits
func ErrExit(w io.Writer, err error) {
	exitCode := ExitCodeErr

	if errors.Is(err, migrate.ErrMigrationSucceeded) {
		// migration success is a good thing. exit with status 0
		printSuccess(w, "migration succeeded, re-run your command to continue")
		os.Exit(ExitCodeOK)
	} else if errors.Is(err, migrate.ErrNeedMigration) {
		exitCode = ExitCodeNeedMigration
	}

	log.Debug(err.Error())
	var qerr qrierr.Error
	if errors.As(err, &qerr) {
		printErr(w, fmt.Errorf(qerr.Message()))
	} else {
		printErr(w, err)
	}
	os.Exit(exitCode)
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

func loadFileIfPath(path string) (data []byte, err error) {
	if path == "" {
		return nil, nil
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadFile(path)
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
