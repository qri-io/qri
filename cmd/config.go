package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/qri/p2p"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	Bootstrap []string `json:"bootstrap"`
}

var defaultCfg = &Config{
	Bootstrap: p2p.DefaultBootstrapAddresses,
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit settings",
	Long:  ``,
}

var configGetCommand = &cobra.Command{
	Use:   "get",
	Short: "Show a configuration setting",
	Run: func(cmd *cobra.Command, args []string) {
		PrintNotYetFinished(cmd)
	},
}

var configSetCommand = &cobra.Command{
	Use:   "set",
	Short: "Set a configuration option",
	Run: func(cmd *cobra.Command, args []string) {
		PrintNotYetFinished(cmd)
	},
}

func init() {
	configCmd.AddCommand(configGetCommand)
	configCmd.AddCommand(configSetCommand)
	RootCmd.AddCommand(configCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() (created bool) {
	var err error
	home := userHomeDir()
	SetNoColor()

	// if cfgFile is specified, override
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		err := viper.ReadInConfig()
		ExitIfErr(err)
		return
	}

	qriPath := os.Getenv("QRI_PATH")
	if qriPath == "" {
		qriPath = filepath.Join(home, ".qri")
	}
	// TODO - this is stupid
	qriPath = strings.Replace(qriPath, "~", home, 1)
	viper.SetDefault(QriRepoPath, filepath.Join(qriPath))
	if err := os.MkdirAll(qriPath, os.ModePerm); err != nil {
		fmt.Errorf("error creating home dir: %s\n", err.Error())
	}

	ipfsFsPath := os.Getenv("IPFS_PATH")
	if ipfsFsPath == "" {
		ipfsFsPath = filepath.Join(home, ".ipfs")
	}
	ipfsFsPath = strings.Replace(ipfsFsPath, "~", home, 1)
	viper.SetDefault(IpfsFsPath, ipfsFsPath)

	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath(qriPath)  // add QRI_PATH env var

	created, err = EnsureConfigFile()
	ExitIfErr(err)

	err = viper.ReadInConfig()
	ExitIfErr(err)

	return
}

func configFilepath() string {
	path := viper.ConfigFileUsed()
	if path == "" {
		path = filepath.Join(viper.GetString(QriRepoPath), "config.json")
	}
	return path
}

func EnsureConfigFile() (bool, error) {
	if _, err := os.Stat(configFilepath()); os.IsNotExist(err) {
		fmt.Println("writing config file")
		return true, WriteConfigFile(defaultCfg)
	}
	return false, nil
}

func WriteConfigFile(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configFilepath(), data, os.ModePerm)
}
