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
	"flag"
	"fmt"
	"path/filepath"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/spf13/cobra"
)

var (
	initIpfsConfigFile string
	// initMetaFile   string
	// initName       string
	// initPassive    bool
	// initRescursive bool
)

// initCmd represents the init command
var initIpfsCmd = &cobra.Command{
	Use:   "init-ipfs",
	Short: "Initialize an ipfs repository",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if initFile == "" {
			ErrExit(fmt.Errorf("please provide a file argument"))
		}

		path, err := filepath.Abs(initFile)
		ExitIfErr(err)

		err = ipfs.InitRepo(path, initIpfsConfigFile)
		ExitIfErr(err)
	},
}

func init() {
	flag.Parse()
	RootCmd.AddCommand(initIpfsCmd)
	initIpfsCmd.Flags().StringVarP(&initIpfsConfigFile, "config", "c", "", "config file for initialization")
}
