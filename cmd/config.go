package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/qri/config"
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
	Annotations: map[string]string{
		"group": "other",
	},
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

		wpk, err := cmd.Flags().GetBool("with-private-keys")
		ExitIfErr(err)
		format, err := cmd.Flags().GetString("format")
		ExitIfErr(err)
		concise, err := cmd.Flags().GetBool("concise")
		ExitIfErr(err)
		output, err := cmd.Flags().GetString("output")
		ExitIfErr(err)

		params := &core.GetConfigParams{
			WithPrivateKey: wpk,
			Format:         format,
			Concise:        concise,
		}

		if len(args) == 1 {
			params.Field = args[0]
		}

		var data []byte

		err = core.GetConfig(params, &data)
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

var configSetCommand = &cobra.Command{
	Use:   "set",
	Short: "Set a configuration option",
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args)%2 != 0 {
			ErrExit(fmt.Errorf("wrong number of arguments. arguments must be in the form: [path value]"))
		}
		ip := config.ImmutablePaths()
		photoPaths := map[string]bool{
			"profile.photo":  true,
			"profile.poster": true,
			"profile.thumb":  true,
		}
		req, err := profileRequests(false)
		ExitIfErr(err)

		for i := 0; i < len(args)-1; i = i + 2 {
			var value interface{}
			path := strings.ToLower(args[i])
			if ip[path] {
				ErrExit(fmt.Errorf("cannot set path %s", path))
			}

			if photoPaths[path] {
				err = setPhotoPath(req, path, args[i+1])
				ExitIfErr(err)
			} else {
				err = yaml.Unmarshal([]byte(args[i+1]), &value)
				ExitIfErr(err)

				err = core.Config.Set(path, value)
				ExitIfErr(err)
			}
		}

		err = core.SetConfig(core.Config)
		ExitIfErr(err)

		printSuccess("config updated")
	},
}

func setPhotoPath(req *core.ProfileRequests, proppath, filepath string) error {
	f, err := loadFileIfPath(filepath)
	if err != nil {
		return err
	}

	p := &core.FileParams{
		Filename: f.Name(),
		Data:     f,
	}
	res := &config.ProfilePod{}

	switch proppath {
	case "profile.photo", "profile.thumb":
		if err := req.SetProfilePhoto(p, res); err != nil {
			return err
		}
	case "profile.poster":
		if err := req.SetPosterPhoto(p, res); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized path to set photo: %s", proppath)
	}

	return nil
}

func init() {
	configGetCommand.Flags().Bool("with-private-keys", false, "include private keys in export")
	configGetCommand.Flags().BoolP("concise", "c", false, "print output without indentation, only applies to json format")
	configGetCommand.Flags().StringP("format", "f", "yaml", "data format to export. either json or yaml")
	configGetCommand.Flags().StringP("output", "o", "", "path to export to")
	configCmd.AddCommand(configGetCommand)

	configCmd.AddCommand(configSetCommand)

	RootCmd.AddCommand(configCmd)
}
