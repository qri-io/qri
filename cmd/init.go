package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	initOverwrite      bool
	initIPFS           bool
	initIPFSConfigFile string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a qri repo",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var cfgData []byte

		if QRIRepoInitialized() && !initOverwrite {
			ErrExit(fmt.Errorf("repo already initialized. use --overwrite to overwrite this repo, erasing all data"))
		}

		// if cfgFile is specified, override
		if cfgFile != "" {
			f, err := os.Open(cfgFile)
			ExitIfErr(err)
			cfgData, err = ioutil.ReadAll(f)
			ExitIfErr(err)
		} else {
			cfgData = defaultCfgBytes()
		}

		cfg := &Config{}
		err := yaml.Unmarshal(cfgData, cfg)
		ExitIfErr(err)

		if err := os.MkdirAll(QriRepoPath, os.ModePerm); err != nil {
			ErrExit(fmt.Errorf("error creating home dir: %s\n", err.Error()))
		}
		err = WriteConfigFile(cfg)
		ExitIfErr(err)

		err = viper.ReadInConfig()
		ExitIfErr(err)

		if initIPFS {
			err := ipfs.InitRepo(IpfsFsPath, initIPFSConfigFile)
			ExitIfErr(err)
		}
	},
}

// QRIRepoInitialized checks to see if a repository has been initialized at $QRI_PATH
func QRIRepoInitialized() bool {
	// for now this just checks for an existing config file
	_, err := os.Stat(configFilepath())
	return !os.IsNotExist(err)
}

func initRepoIfEmpty(repoPath, configPath string) error {
	if repoPath != "" {
		if _, err := os.Stat(filepath.Join(repoPath, "config")); os.IsNotExist(err) {
			if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
				return err
			}
			if err := ipfs.InitRepo(repoPath, configPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	RootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&initOverwrite, "overwrite", "", false, "overwrite repo if one exists")
	initCmd.Flags().BoolVarP(&initIPFS, "init-ipfs", "", true, "initialize an IPFS repo if one isn't present")
	initCmd.Flags().StringVarP(&initIPFSConfigFile, "ipfs-config", "", "", "config file for initialization")
}
