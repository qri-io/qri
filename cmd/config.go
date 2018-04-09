package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/qri/core"
	"io/ioutil"
	"os"
	"path/filepath"

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

var configGetCommand = &cobra.Command{
	Use:   "get",
	Short: "Show a configuration setting",
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		for _, path := range args {
			value, err := core.Config.Get(path)
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

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "export configuration settings",
	Long: `export outputs your current configuration file with private keys 
removed by default.
export makes it easier to share your qri configuration settings.

We've added the --with-private-keys option to include private keys in the export
PLEASE PLEASE PLEASE NEVER SHARE YOUR PRIVATE KEYS WITH ANYONE. EVER.
Anyone with your private keys can impersonate you on qri.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			data []byte
			err  error
			cfg  = core.Config
		)

		wpk, err := cmd.Flags().GetBool("with-private-keys")
		ExitIfErr(err)
		format, err := cmd.Flags().GetString("format")
		ExitIfErr(err)
		pretty, err := cmd.Flags().GetBool("pretty")
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

		switch format {
		case "json":
			if pretty {
				data, err = json.MarshalIndent(cfg, "", "  ")
			} else {
				data, err = json.Marshal(cfg)
			}
		case "yaml":
			data, err = yaml.Marshal(cfg)
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

	configExportCmd.Flags().Bool("with-private-keys", false, "include private keys in export")
	configExportCmd.Flags().BoolP("pretty", "p", false, "pretty-print output")
	configExportCmd.Flags().StringP("format", "f", "json", "data format to export. either json or yaml")
	configExportCmd.Flags().StringP("output", "o", "", "path to export to")
	configCmd.AddCommand(configExportCmd)

	RootCmd.AddCommand(configCmd)
}
