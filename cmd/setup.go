package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	// config "gx/ipfs/QmViBzgruNUoLNBnXcx8YWbDNwV8MNGEGKkLo6JGetygdw/go-ipfs/repo/config"
)

var (
	setupOverwrite      bool
	setupIPFS           bool
	setupIPFSConfigFile string
	setupConfigData     string
	setupProfileData    string
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initialize qri and IPFS repositories, provision a new qri ID",
	Long: `
Setup is the first command you run to get a fresh install of qri. If you’ve 
never run qri before, you’ll need to run setup before you can do anything. 

Setup does a few things:
- create a qri repository to keep all of your data
- provisions a new qri ID
- create an IPFS repository if one doesn’t exist

This command is automatically run if you invoke any qri command without first 
running setup. If setup has already been run, by default qri won’t let you 
overwrite this info.`,
	Run: func(cmd *cobra.Command, args []string) {
		var cfgData []byte

		if QRIRepoInitialized() && !setupOverwrite {
			// use --overwrite to overwrite this repo, erasing all data and deleting your account for good
			// this is usually a terrible idea
			ErrExit(fmt.Errorf("repo already initialized"))
		}
		fmt.Println("initializing qri repo")

		envVars := map[string]*string{
			"QRI_SETUP_CONFIG_DATA":  &setupConfigData,
			"QRI_SETUP_PROFILE_DATA": &setupProfileData,
		}
		mapEnvVars(envVars)

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

		if setupConfigData != "" {
			err = readAtFile(&setupConfigData)
			ExitIfErr(err)
			err = json.Unmarshal([]byte(setupConfigData), cfg)
			ExitIfErr(err)
		}

		err = cfg.ensurePrivateKey()
		ExitIfErr(err)

		if err := os.MkdirAll(QriRepoPath, os.ModePerm); err != nil {
			ErrExit(fmt.Errorf("error creating home dir: %s", err.Error()))
		}
		err = writeConfigFile(cfg)
		ExitIfErr(err)

		err = viper.ReadInConfig()
		ExitIfErr(err)

		if setupIPFS {
			err = ipfs.InitRepo(IpfsFsPath, setupIPFSConfigFile)
			if err != nil && strings.Contains(err.Error(), "already") {
				err = nil
			}
			ExitIfErr(err)
		}

		if setupProfileData != "" {
			err = readAtFile(&setupProfileData)
			ExitIfErr(err)

			p := &core.Profile{}
			err = json.Unmarshal([]byte(setupProfileData), p)
			ExitIfErr(err)

			pr, err := profileRequests(false)
			ExitIfErr(err)

			res := &core.Profile{}
			err = pr.SaveProfile(p, res)
			ExitIfErr(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(setupCmd)
	setupCmd.Flags().BoolVarP(&setupOverwrite, "overwrite", "", false, "overwrite repo if one exists")
	setupCmd.Flags().BoolVarP(&setupIPFS, "init-ipfs", "", true, "initialize an IPFS repo if one isn't present")
	setupCmd.Flags().StringVarP(&setupIPFSConfigFile, "ipfs-config", "", "", "config file for initialization")
	setupCmd.Flags().StringVarP(&setupConfigData, "id", "", "", "json-encoded configuration data, specify a filepath with '@' prefix")
	setupCmd.Flags().StringVarP(&setupProfileData, "profile", "", "", "json-encoded user profile data, specify a filepath with '@' prefix")
}

// QRIRepoInitialized checks to see if a repository has been initialized at $QRI_PATH
func QRIRepoInitialized() bool {
	// for now this just checks for an existing config file
	_, err := os.Stat(configFilepath())
	return !os.IsNotExist(err)
}

func mapEnvVars(vars map[string]*string) {
	for envVar, value := range vars {
		envVal := os.Getenv(envVar)
		if envVal != "" {
			fmt.Printf("reading %s from env\n", envVar)
			*value = envVal
		}
	}
}

func setupRepoIfEmpty(repoPath, configPath string) error {
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

// readAtFile is a unix curl inspired method. any data input that begins with "@"
// is assumed to instead be a filepath that should be read & replaced with the contents
// of the specified path
func readAtFile(data *string) error {
	d := *data
	if len(d) > 0 && d[0] == '@' {
		fileData, err := ioutil.ReadFile(d[1:])
		if err != nil {
			return err
		}
		*data = string(fileData)
	}
	return nil
}
