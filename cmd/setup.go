package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/doggos"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

var (
	setupOverwrite      bool
	setupIPFS           bool
	setupPeername       string
	setupIPFSConfigData string
	setupConfigData     string
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "initialize qri and IPFS repositories, provision a new qri ID",
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
	Example: `  run setup with a peername of your choosing:
	$ qri setup --peername=your_great_peername`,
	Run: func(cmd *cobra.Command, args []string) {
		// var cfgData []byte
		var cfg *config.Config

		if QRIRepoInitialized() && !setupOverwrite {
			// use --overwrite to overwrite this repo, erasing all data and deleting your account for good
			// this is usually a terrible idea
			ErrExit(fmt.Errorf("repo already initialized"))
		}
		fmt.Printf("setting up qri repo at: %s\n", QriRepoPath)

		cfg = config.DefaultConfig()

		envVars := map[string]*string{
			"QRI_SETUP_CONFIG_DATA":      &setupConfigData,
			"QRI_SETUP_IPFS_CONFIG_DATA": &setupIPFSConfigData,
		}
		mapEnvVars(envVars)

		// if cfgFile is specified, override
		// if cfgFile != "" {
		// 	f, err := os.Open(cfgFile)
		// 	ExitIfErr(err)
		// 	cfgData, err = ioutil.ReadAll(f)
		// 	ExitIfErr(err)
		// } else {
		// 	cfgData, _ = yaml.Marshal(config.Config{}.Default())
		// }

		if setupConfigData != "" {
			err := readAtFile(&setupConfigData)
			ExitIfErr(err)
			err = json.Unmarshal([]byte(setupConfigData), cfg)
			if cfg.Profile != nil {
				setupPeername = cfg.Profile.Peername
			}
			ExitIfErr(err)
		}

		if cfg.Profile == nil {
			cfg.Profile = config.DefaultProfile()
		}
		anon, err := cmd.Flags().GetBool("anonymous")
		ExitIfErr(err)

		if setupPeername != "" {
			cfg.Profile.Peername = setupPeername
		} else if cfg.Profile.Peername == doggos.DoggoNick(cfg.Profile.ID) && !anon {
			cfg.Profile.Peername = inputText("choose a peername:", doggos.DoggoNick(cfg.Profile.ID))
			printSuccess(cfg.Profile.Peername)
		}

		// TODO - should include a call to config.Validate here once config has a validate function

		if err := os.MkdirAll(QriRepoPath, os.ModePerm); err != nil {
			ErrExit(fmt.Errorf("error creating home dir: %s", err.Error()))
		}

		if setupIPFS {

			tmpIPFSConfigPath := ""
			if setupIPFSConfigData != "" {
				err = readAtFile(&setupIPFSConfigData)
				ExitIfErr(err)

				// TODO - remove this temp file & instead adjust ipfs.InitRepo to accept an io.Reader
				tmpIPFSConfigPath = filepath.Join(os.TempDir(), "ipfs_init_config")

				err = ioutil.WriteFile(tmpIPFSConfigPath, []byte(setupIPFSConfigData), os.ModePerm)
				ExitIfErr(err)

				defer func() {
					os.Remove(tmpIPFSConfigPath)
				}()
			}

			err = ipfs.InitRepo(IpfsFsPath, tmpIPFSConfigPath)
			if err != nil && strings.Contains(err.Error(), "already") {
				err = nil
			}
			ExitIfErr(err)
		} else if _, err := os.Stat(IpfsFsPath); os.IsNotExist(err) {
			printWarning("no IPFS repo exists at %s, things aren't going to work properly", IpfsFsPath)
		}

		err = cfg.WriteToFile(configFilepath())
		ExitIfErr(err)

		core.Config = cfg
	},
}

func init() {
	RootCmd.AddCommand(setupCmd)
	setupCmd.Flags().BoolP("anonymous", "a", false, "use an auto-generated peername")
	setupCmd.Flags().BoolVarP(&setupOverwrite, "overwrite", "", false, "overwrite repo if one exists")
	setupCmd.Flags().BoolVarP(&setupIPFS, "init-ipfs", "", true, "initialize an IPFS repo if one isn't present")
	setupCmd.Flags().StringVarP(&setupPeername, "peername", "", "", "choose your desired peername")
	setupCmd.Flags().StringVarP(&setupIPFSConfigData, "ipfs-config", "", "", "json-encoded configuration data, specify a filepath with '@' prefix")
	setupCmd.Flags().StringVarP(&setupConfigData, "conifg-data", "", "", "json-encoded configuration data, specify a filepath with '@' prefix")
	// setupCmd.Flags().StringVarP(&setupProfileData, "profile", "", "", "json-encoded user profile data, specify a filepath with '@' prefix")
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
