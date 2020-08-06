package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo/gen"
	"github.com/spf13/cobra"
)

// NewSetupCommand creates a setup command
func NewSetupCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &SetupOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "initialize qri and IPFS repositories, provision a new qri ID",
		Long: `Setup is the first command you run to get a fresh install of Qri. If you’ve 
never run qri before, you’ll need to run setup before you can do anything. 

Setup does a few things:
- create a qri repository to keep all of your data
- provisions a new qri ID
- create an IPFS repository if one doesn’t exist

This command is automatically run if you invoke any Qri command without first 
running setup. If setup has already been run, by default Qri won’t let you 
overwrite this info.

Use the ` + "`--remove`" + ` to remove your Qri repo. This deletes your entire repo, 
including all your datasets, and de-registers your username from the registry.`,
		Example: `  # Run setup with a username of your choosing:
  $ qri setup --username=your_great_username`,
		Annotations: map[string]string{
			"group": "other",
		},
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run(f)
		},
	}

	cmd.Flags().BoolVarP(&o.Anonymous, "anonymous", "a", false, "use an auto-generated username")
	cmd.Flags().BoolVarP(&o.Overwrite, "overwrite", "", false, "overwrite repo if one exists")
	cmd.Flags().BoolVarP(&o.IPFS, "init-ipfs", "", true, "initialize an IPFS repo if one isn't present")
	cmd.Flags().BoolVarP(&o.Remove, "remove", "", false, "permanently remove qri, overrides all setup options")
	cmd.Flags().StringVarP(&o.Registry, "registry", "", "", "override default registry URL, set to 'none' to remove registry")
	cmd.Flags().StringVarP(&o.Username, "username", "", "", "choose your desired username")
	cmd.Flags().StringVarP(&o.IPFSConfigData, "ipfs-config", "", "", "json-encoded configuration data, specify a filepath with '@' prefix")
	cmd.Flags().StringVarP(&o.ConfigData, "config-data", "", "", "json-encoded configuration data, specify a filepath with '@' prefix")
	cmd.Flags().BoolVar(&o.GimmeDoggo, "gimme-doggo", false, "create and display a doggo name only")

	return cmd
}

// SetupOptions encapsulates state for the setup command
type SetupOptions struct {
	ioes.IOStreams
	repoPath  string
	Generator gen.CryptoGenerator

	Anonymous      bool
	Overwrite      bool
	IPFS           bool
	Remove         bool
	Username       string
	Registry       string
	IPFSConfigData string
	ConfigData     string
	GimmeDoggo     bool
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *SetupOptions) Complete(f Factory, args []string) (err error) {
	o.repoPath = f.RepoPath()
	o.Generator = f.CryptoGenerator()
	return
}

// Run executes the setup command
func (o *SetupOptions) Run(f Factory) error {
	if o.GimmeDoggo {
		return o.CreateAndDisplayDoggo()
	}

	if o.Remove {
		cfg, err := f.Config()
		if err != nil {
			return err
		}
		// TODO - add a big warning here that requires user input
		err = lib.Teardown(lib.TeardownParams{
			Config:   cfg,
			RepoPath: o.repoPath,
		})
		if err != nil {
			return err
		}
		printSuccess(o.Out, "repo removed")
		return nil
	}

	if err := o.DoSetup(f); err != nil {
		return err
	}

	printSuccess(o.Out, "set up qri repo at: %s\n", o.repoPath)
	return nil
}

// DoSetup executes the setup-ie bit from the setup command
func (o *SetupOptions) DoSetup(f Factory) (err error) {
	cfg := config.DefaultConfig()

	envVars := map[string]*string{
		"QRI_SETUP_CONFIG_DATA":      &o.ConfigData,
		"QRI_SETUP_IPFS_CONFIG_DATA": &o.IPFSConfigData,
	}
	mapEnvVars(envVars)

	if o.ConfigData != "" {
		if err = readAtFile(&o.ConfigData); err != nil {
			return err
		}

		err = json.Unmarshal([]byte(o.ConfigData), cfg)
		if cfg.Profile != nil {
			o.Username = cfg.Profile.Peername
		}
		if err != nil {
			return err
		}
	}

	// Handle the --username flag, or prompt the user for a username
	if o.Username != "" {
		cfg.Profile.Peername = o.Username
	} else if !o.Anonymous {
		cfg.Profile.Peername = inputText(o.Out, o.In, "choose username (leave empty to generate a default name):", "")
	}

	// If a username was passed with the --username flag or entered by prompt, make sure its valid
	if cfg.Profile.Peername != "" {
		if err := dsref.EnsureValidUsername(cfg.Profile.Peername); err != nil {
			return err
		}
	}

	if o.Registry == "none" {
		cfg.Registry = nil
	} else if o.Registry != "" {
		cfg.Registry.Location = o.Registry
	}

	p := lib.SetupParams{
		Config:    cfg,
		RepoPath:  o.repoPath,
		SetupIPFS: o.IPFS,
		Register:  o.Registry == "none",
		Generator: o.Generator,
	}

	if o.IPFSConfigData != "" {
		if err = readAtFile(&o.IPFSConfigData); err != nil {
			return err
		}
		p.SetupIPFSConfigData = []byte(o.IPFSConfigData)
	}

	return lib.Setup(p)
}

// CreateAndDisplayDoggo creates and display a doggo name
func (o *SetupOptions) CreateAndDisplayDoggo() error {
	_, peerID := o.Generator.GeneratePrivateKeyAndPeerID()
	dognick := o.Generator.GenerateNickname(peerID)
	printSuccess(o.Out, dognick)
	return nil
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

func setupRepoIfEmpty(repoPath, configPath string, g gen.CryptoGenerator) error {
	if repoPath != "" {
		if _, err := os.Stat(filepath.Join(repoPath, "config")); os.IsNotExist(err) {
			if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
				return err
			}
			if err := g.GenerateEmptyIpfsRepo(repoPath, configPath); err != nil {
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
