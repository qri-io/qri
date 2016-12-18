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
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/spf13/cobra"
)

// pathCmd represents the path command
var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the path or url of a dataset's address",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		adr := GetAddress(cmd, args)
		fmt.Println(filepath.Join(cachePath(), DatasetPath(&dataset.Dataset{Address: adr})))
	},
}

func init() {
	RootCmd.AddCommand(pathCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pathCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// pathCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
