package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var (
	// QriRepoPath is the path to the QRI repository
	QriRepoPath string
	// IpfsFsPath is the path to the IPFS repo
	IpfsFsPath string
	// cfgFile overrides default configuration file with a custom filepath
	cfgFile string
	// setting noLoadConfig to true will skip the the default call to LoadConfig
	noLoadConfig bool
	// global pagination variables
	pageNum  int
	pageSize int
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "qri",
	Short: "qri GDVCS CLI",
	Long: `
qri ("query") is a global dataset version control system 
on the distributed web.

https://qri.io

Feedback, questions, bug reports, and contributions are welcome!
https://github.com/qri-io/qri/issues`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Catch errors & pretty-print.
	// comment this out to get stack traces back.
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				fmt.Println(err.Error())
			} else {
				fmt.Println(r)
			}
		}
	}()

	if err := RootCmd.Execute(); err != nil {
		printErr(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initializeCLI)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $QRI_PATH/config.yaml)")
	// RootCmd.PersistentFlags().BoolVarP(&noColor, "no-color", "c", false, "disable colorized output")
}

// initializeCLI sets up the CLI, reading in config file and ENV variables if set.
func initializeCLI() {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	QriRepoPath = os.Getenv("QRI_PATH")
	if QriRepoPath == "" {
		QriRepoPath = filepath.Join(home, ".qri")
	}

	IpfsFsPath = os.Getenv("IPFS_PATH")
	if IpfsFsPath == "" {
		IpfsFsPath = filepath.Join(home, ".ipfs")
	}

	return
}
