package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func configFilepath() string {
	if cfgFile == "" {
		return filepath.Join(QriRepoPath, "config.yaml")
	}
	return cfgFile
}

func loadConfig() {
	core.ConfigFilepath = configFilepath()
	if err := core.LoadConfig(core.ConfigFilepath); err != nil {
		ErrExit(err)
	}
}

// configCmd represents commands that read & modify configuration settings
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "get and set local configuration information",
	Long: `
config encapsulates all settings that control the behaviour of qri.
This includes all kinds of stuff: your profile details; enabling & disabling 
different services; what kind of output qri logs to; 
which ports on qri serves on; etc.

Configuration is stored as a .yaml file kept at $QRI_PATH, or provided at CLI 
runtime via command a line argument.`,
	Example: `  # get your profile information
  $ qri config get profile

  # set your api port to 4444
  $ qri config set api.port 4444

  # disable rpc connections:
  $ qri config set rpc.enabled false`,
}

var configSetCommand = &cobra.Command{
	Use:   "set",
	Short: "Set a configuration option",
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		// var err error
		if len(args)%2 != 0 {
			ErrExit(fmt.Errorf("wrong number of arguments. arguments must be in the form: [path value]"))
		}

		for i := 0; i < len(args)-1; i = i + 2 {
			var value interface{}
			err := yaml.Unmarshal([]byte(args[i+1]), &value)
			ExitIfErr(err)

			err = core.Config.Set(args[i], value)
			ExitIfErr(err)
		}

		err := core.Config.WriteToFile(core.ConfigFilepath)
		ExitIfErr(err)
		printSuccess("config updated")
	},
}

var configGetCommand = &cobra.Command{
	Use:   "get",
	Short: "get configuration settings",
	Long: `get outputs your current configuration file with private keys 
removed by default, making it easier to share your qri configuration settings.

The --with-private-keys option will show private keys.
PLEASE PLEASE PLEASE NEVER SHARE YOUR PRIVATE KEYS WITH ANYONE. EVER.
Anyone with your private keys can impersonate you on qri.`,
	Args: cobra.MaximumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			data   []byte
			err    error
			cfg    = core.Config
			encode interface{}
		)

		wpk, err := cmd.Flags().GetBool("with-private-keys")
		ExitIfErr(err)
		format, err := cmd.Flags().GetString("format")
		ExitIfErr(err)
		concise, err := cmd.Flags().GetBool("concise")
		ExitIfErr(err)
		output, err := cmd.Flags().GetString("output")
		ExitIfErr(err)

		if !wpk {
			if cfg.Profile != nil {
				cfg.Profile.PrivKey = ""
			}
			if cfg.P2P != nil {
				cfg.P2P.PrivKey = ""
			}
		}

		if len(args) == 1 {
			encode, err = cfg.Get(args[0])
			ExitIfErr(err)
		} else {
			encode = cfg
		}

		switch format {
		case "json":
			if concise {
				data, err = json.Marshal(encode)
			} else {
				data, err = json.MarshalIndent(encode, "", "  ")
			}
		case "yaml":
			data, err = yaml.Marshal(encode)
		}
		ExitIfErr(err)

		if output != "" {
			err = ioutil.WriteFile(output, data, os.ModePerm)
			ExitIfErr(err)
			printSuccess("config file written to: %s", output)
			return
		}

		fmt.Println(string(data))
	},
}

func init() {
	configCmd.AddCommand(configGetCommand)
	configCmd.AddCommand(configSetCommand)

	configGetCommand.Flags().Bool("with-private-keys", false, "include private keys in export")
	configGetCommand.Flags().BoolP("concise", "c", false, "print output without indentation, only applies to json format")
	configGetCommand.Flags().StringP("format", "f", "json", "data format to export. either json or yaml")
	configGetCommand.Flags().StringP("output", "o", "", "path to export to")
	configCmd.AddCommand(configGetCommand)

	RootCmd.AddCommand(configCmd)
}
