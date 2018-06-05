package cmd

import (
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
	noPrompt     bool
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

func init() {
	cobra.OnInitialize(initializeCLI)
	// TODO: write a test that verifies this works with our new yaml config
	// RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $QRI_PATH/config.yaml)")
	// RootCmd.PersistentFlags().BoolVarP(&noColor, "no-color", "c", false, "disable colorized output")
	RootCmd.SetUsageTemplate(rootUsageTemplate)
	RootCmd.PersistentFlags().BoolVarP(&noPrompt, "no-prompt", "", false, "disable all interactive prompts")
	for _, cmd := range RootCmd.Commands() {
		cmd.SetUsageTemplate(defaultUsageTemplate)
	}
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
