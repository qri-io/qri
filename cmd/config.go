package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/qri-io/qri/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	// cfg is the global configuration object for the CLI
	cfg *config.Config
	// setting ignoreCfg to true will prevent loadConfig from doing anything
	ignoreCfg bool
)

// configCmd represents commands that read & modify configuration settings
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "get and set local configuration information",
	Long: `
config is a bit of a cheat right now. we’re going to break profile out into 
proper parameter-based commands later on, but for now we’re hoping you can edit 
a YAML file of configuration information. 

For now running qri config get will write a file called config.yaml containing 
current configuration info. Edit that file and run config set <file> to write 
configuration details back.

Expect the config command to change in future releases.`,
}

var (
	getConfigFilepath, setConfigFilepath string
)

var configGetCommand = &cobra.Command{
	Use:   "get",
	Short: "Show a configuration setting",
	Run: func(cmd *cobra.Command, args []string) {
		for _, path := range args {
			value, err := cfg.Get(path)
			ExitIfErr(err)
			data, err := yaml.Marshal(value)
			ExitIfErr(err)
			printSuccess(string(data))
		}
	},
}

var configSetCommand = &cobra.Command{
	Use:   "set",
	Short: "Set a configuration option",
	Run: func(cmd *cobra.Command, args []string) {
		// var err error

		if len(args)%2 != 0 {
			ErrExit(fmt.Errorf("wrong number of arguments. arguments must be in the form: [path value]"))
		}

		for i := 0; i < len(args)-1; i = i + 2 {
			var value interface{}
			err := yaml.Unmarshal([]byte(args[i+1]), &value)
			ExitIfErr(err)

			err = cfg.Set(args[i], value)
			ExitIfErr(err)
		}

		err := cfg.WriteToFile(configFilepath())
		ExitIfErr(err)
		printSuccess("config updated")
	},
}

func configFilepath() string {
	return filepath.Join(QriRepoPath, "config.yaml")
}

func init() {
	configGetCommand.Flags().StringVarP(&getConfigFilepath, "file", "f", "", "file to save YAML config info to")
	configCmd.AddCommand(configGetCommand)

	configSetCommand.Flags().StringVarP(&setConfigFilepath, "file", "f", "", "filepath to *complete* yaml config info file")
	configCmd.AddCommand(configSetCommand)

	RootCmd.AddCommand(configCmd)
}

func loadConfig() (err error) {
	if ignoreCfg {
		return nil
	}

	cfg, err = config.ReadFromFile(configFilepath())

	if err == nil && cfg.Profile == nil {
		err = fmt.Errorf("missing profile")
	}

	if err != nil {
		str := `couldn't read config file. error
	%s
if you've recently updated qri your config file may no longer be valid.
The easiest way to fix this is to delete your repository at:
	%s
and start with a fresh qri install by running 'qri setup' again.
Sorry, we know this is not exactly a great experience, from this point forward
we won't be shipping changes that require starting over.
`
		err = fmt.Errorf(str, err.Error(), QriRepoPath)
	}

	return err
}
