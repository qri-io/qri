package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

const (
	QriRepoPath = "QriRepoPath"
	IpfsFsPath  = "IpfsFsPath"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "qri",
	Short: "qri.io command line client",
	Long: `this is a very early tool for working with datasets on the distributed web.
	At the moment it's a bit an experiment.

	Email brendan with any questions:
	sparkle_pony_2000@qri.io`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		PrintErr(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $QRI_PATH/config.json)")
	RootCmd.PersistentFlags().BoolVarP(&noColor, "no-color", "c", false, "disable colorized output")
}
