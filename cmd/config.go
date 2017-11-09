package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
)

type Config struct {
	Bootstrap []string
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit settings",
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

func init() {
	configCmd.AddCommand(configGetCommand)
	configCmd.AddCommand(configSetCommand)
	RootCmd.AddCommand(configCmd)
}

func WriteConfigFile(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fmt.Sprintf("%s/.qri.json", os.Getenv("HOME")), data, os.ModePerm)
}
