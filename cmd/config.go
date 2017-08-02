// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
)

type Config struct {
	// Remotes []*Remote `json:"remotes"`
	// Folders []*Folder `json:"folders"`
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
