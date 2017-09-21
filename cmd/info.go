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
	"fmt"

	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"describe"},
	Short:   "Show info about a dataset",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(args)
		if len(args) == 0 {
			fmt.Println("please specify an address to get the info of")
			return
		}

		// ds, err := GetNamespaces(cmd, args).Dataset(dataset.NewAddress(args[0]))
		// ExitIfErr(err)
		// PrintDatasetDetailedInfo(ds)
	},
}

func init() {
	// RootCmd.AddCommand(infoCmd)
}
