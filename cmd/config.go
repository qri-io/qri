package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/qri/p2p"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// Config configures the behavior of qri
type Config struct {
	// Initialized is a flag for when this repo has been properly initialized at least once.
	// used to check weather default datasets should be added or not
	Initialized bool
	// Identity Configuration details
	// Identity IdentityCfg
	// List of nodes to boostrap to
	Bootstrap []string
	// Datastore configuration details
	// Datastore       DatastoreCfg
	// DefaultDatasets is a list of datasets to grab on initially joining the network
	DefaultDatasets map[string]string
}

// IdentityCfg holds details about user identity & configuration
// type IdentityCfg struct {
// 	// ID to feed to IPFS node, and for profile identification
// 	PeerID string
// 	// PrivateKey for
// 	PrivateKey string
// 	// Profile
// 	Profile *core.Profile
// }

// DatastoreCfg configures the underlying IPFS datastore. WIP.
// type DatastoreCfg struct {
// 	StorageMax         string
// 	StorageGCWatermark int
// 	GCPeriod           string
// }

func defaultCfgBytes() []byte {
	cfg := &Config{
		Initialized: false,
		Bootstrap:   p2p.DefaultBootstrapAddresses,
		// defaultDatasets is a hard-coded dataset added when a new qri repo is created
		// these hashes should always/highly available
		DefaultDatasets: map[string]string{
			// fivethirtyeight comic characters
			"comic_characters": "/ipfs/QmcqkHFA2LujZxY38dYZKmxsUstN4unk95azBjwEhwrnM6/dataset.json",
		},
	}

	data, err := yaml.Marshal(cfg)
	ExitIfErr(err)
	return data
}

// configCmd represents commands that read & modify configuration settings
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Read & Edit settings",
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

func configFilepath() string {
	// path := viper.ConfigFileUsed()
	// if path == "" {
	// path = filepath.Join(QriRepoPath, "config.yaml")
	// }
	return filepath.Join(QriRepoPath, "config.yaml")
}

func ReadConfigFile() (*Config, error) {
	data, err := ioutil.ReadFile(configFilepath())
	if err != nil {
		return nil, err
	}
	cfg := Config{}
	err = yaml.Unmarshal(data, &cfg)
	return &cfg, err
}

func WriteConfigFile(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configFilepath(), data, os.ModePerm)
}

func loadConfig() {
	viper.SetConfigName("config")    // name of config file (without extension)
	viper.AddConfigPath(QriRepoPath) // add QRI_PATH env var
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("QRI_")

	// if cfgFile is specified, override
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}
	viper.ReadInConfig()
	// ExitIfErr(err)
}

func init() {
	configCmd.AddCommand(configGetCommand)
	configCmd.AddCommand(configSetCommand)
	RootCmd.AddCommand(configCmd)
}
