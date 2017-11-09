package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	home := userHomeDir()
	SetNoColor()

	// if cfgFile is specified, override
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		err := viper.ReadInConfig()
		ExitIfErr(err)
		return
	}

	// if err := os.Mkdir(filepath.Join(userHomeDir(), ".qri"), os.ModePerm); err != nil {
	// 	fmt.Errorf("error creating home dir: %s\n", err.Error())
	// }
	qriPath := os.Getenv("QRI_PATH")
	if qriPath == "" {
		qriPath = filepath.Join(home, "qri")
	}

	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath(qriPath)  // add QRI_PATH env var
	// viper.AddConfigPath("$HOME/.qri/config") // adding home directory as first search path
	// viper.AddConfigPath(".")                 // adding home directory as first search path
	viper.AutomaticEnv() // read in environment variables that match

	// TODO - this is stupid
	qriPath = strings.Replace(qriPath, "~", home, 1)
	viper.SetDefault(QriRepoPath, qriPath)

	ipfsFsPath := os.Getenv("IPFS_PATH")
	if ipfsFsPath == "" {
		ipfsFsPath = "$HOME/.ipfs"
	}
	ipfsFsPath = strings.Replace(ipfsFsPath, "~", home, 1)
	viper.SetDefault(IpfsFsPath, ipfsFsPath)

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
