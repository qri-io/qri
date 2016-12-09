package cmd

import (
	"fmt"
	"os"

	"github.com/qri-io/dataset"
	"github.com/qri-io/fs"
	"github.com/qri-io/fs/local"
	lns "github.com/qri-io/namespace/local"
	"github.com/qri-io/namespace/remote"
	"github.com/spf13/cobra"
)

// ErrExit writes an error to stdout & exits
func ErrExit(err error) {
	PrintErr(err)
	os.Exit(1)
}

func ExitIfErr(err error) {
	if err != nil {
		PrintErr(err)
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

func GetAddress(cmd *cobra.Command, args []string) dataset.Address {
	adr := dataset.NewAddress("")
	if len(args) > 0 {
		adr = dataset.NewAddress(args[0])
	}
	return adr
}

// Store creates the appropriate store for a given command
// defaulting to creating a new store from the local directory
func Store(cmd *cobra.Command, args []string) fs.Store {
	return local.NewLocalStore(GetWd())
}

// Cache is the place to put downloaded stuff. default is the local store
func Cache() fs.Store {
	return local.NewLocalStore(GetWd())
}

// Namespaces reads the list of namespaces from the config
func GetNamespaces(cmd *cobra.Command, args []string) Namespaces {
	return Namespaces{
		lns.NewNamespaceFromPath(GetWd()),
		remote.New("localhost", "qri"),
	}
}
