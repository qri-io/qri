// Copyright © 2016 qri.io <info@qri.io>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	Long: `this is a very early (like 0.0.0.0.0.1.alpha) tool for working with datasets.
	At the moment it's a bit an experiment.
	A nice web introduction is available at:
	http://docs.qri.io

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

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.qri.json)")
	RootCmd.PersistentFlags().BoolVarP(&noColor, "no-color", "c", false, "disable colorized output")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	home := userHomeDir()
	SetNoColor()

	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	// if err := os.Mkdir(filepath.Join(userHomeDir(), ".qri"), os.ModePerm); err != nil {
	// 	fmt.Errorf("error creating home dir: %s\n", err.Error())
	// }

	// viper.SetConfigName("config") // name of config file (without extension)
	// // viper.AddConfigPath("$QRI_PATH")  // add QRI_PATH env var
	// viper.AddConfigPath("$HOME/.qri") // adding home directory as first search path
	// viper.AddConfigPath(".")          // adding home directory as first search path
	// viper.AutomaticEnv()              // read in environment variables that match

	qriPath := os.Getenv("QRI_PATH")
	if qriPath == "" {
		qriPath = filepath.Join(home, "qri")
	}
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
	// if err := viper.ReadInConfig(); err == nil {
	// 	// fmt.Println("Using config file:", viper.ConfigFileUsed())
	// }
}
