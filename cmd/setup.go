package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/doggos"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

var (
	setupAnonymous      bool
	setupOverwrite      bool
	setupIPFS           bool
	setupRemove         bool
	setupPeername       string
	setupRegistry       string
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

		if setupRemove {
			loadConfig()
			// TODO - add a big warning here that requires user input
			err := core.Teardown(core.TeardownParams{
				Config:      core.Config,
				QriRepoPath: QriRepoPath,
			})
			ExitIfErr(err)
			printSuccess("repo removed")
			return
		}

		if QRIRepoInitialized() && !setupOverwrite {
			// use --overwrite to overwrite this repo, erasing all data and deleting your account for good
			// this is usually a terrible idea
			ErrExit(fmt.Errorf("repo already initialized"))
		}

		err := doSetup(setupConfigData, setupIPFSConfigData, setupRegistry, setupAnonymous)
		ExitIfErr(err)

		printSuccess("set up qri repo at: %s\n", QriRepoPath)
	},
}

func init() {
	RootCmd.AddCommand(setupCmd)
	setupCmd.Flags().BoolVarP(&setupAnonymous, "anonymous", "a", false, "use an auto-generated peername")
	setupCmd.Flags().BoolVarP(&setupOverwrite, "overwrite", "", false, "overwrite repo if one exists")
	setupCmd.Flags().BoolVarP(&setupIPFS, "init-ipfs", "", true, "initialize an IPFS repo if one isn't present")
	setupCmd.Flags().BoolVarP(&setupRemove, "remove", "", false, "permanently remove qri, overrides all setup options")
	setupCmd.Flags().StringVarP(&setupRegistry, "registry", "", "", "override default registry URL")
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

func doSetup(configData, IPFSConfigData, registry string, anon bool) (err error) {
	cfg := config.DefaultConfig()

	envVars := map[string]*string{
		"QRI_SETUP_CONFIG_DATA":      &configData,
		"QRI_SETUP_IPFS_CONFIG_DATA": &IPFSConfigData,
	}
	mapEnvVars(envVars)

	if configData != "" {
		if err = readAtFile(&configData); err != nil {
			return err
		}

		err = json.Unmarshal([]byte(configData), cfg)
		if cfg.Profile != nil {
			setupPeername = cfg.Profile.Peername
		}
		if err != nil {
			return err
		}
	}

	if cfg.Profile == nil {
		cfg.Profile = config.DefaultProfile()
	}

	if setupPeername != "" {
		cfg.Profile.Peername = setupPeername
	} else if cfg.Profile.Peername == doggos.DoggoNick(cfg.Profile.ID) && !anon {
		cfg.Profile.Peername = inputText("choose a peername:", doggos.DoggoNick(cfg.Profile.ID))
	}

	if registry != "" {
		cfg.Registry.Location = registry
	}

	p := core.SetupParams{
		Config:         cfg,
		QriRepoPath:    QriRepoPath,
		ConfigFilepath: configFilepath(),
		SetupIPFS:      setupIPFS,
		IPFSFsPath:     IpfsFsPath,
	}

	if IPFSConfigData != "" {
		err = readAtFile(&IPFSConfigData)
		ExitIfErr(err)
		p.SetupIPFSConfigData = []byte(IPFSConfigData)
	}

	for {
		err := core.Setup(p)
		if err != nil {
			if err == core.ErrHandleTaken {
				printWarning("peername '%s' already taken", cfg.Profile.Peername)
				cfg.Profile.Peername = inputText("choose a peername:", doggos.DoggoNick(cfg.Profile.ID))
				continue
			} else {
				ErrExit(err)
			}
		}
		break
	}

	return nil
}
