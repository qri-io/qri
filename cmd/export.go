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
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dsfs"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "export dataset data",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("please specify a dataset name to export")
			return
		}
		path := cmd.Flag("output").Value.String()
		if path == "" {
			// TODO - support printing to stdout
			ErrExit(fmt.Errorf("please specify an output path"))
		}

		r := GetRepo()
		store, err := GetIpfsFilestore()
		ExitIfErr(err)

		ds, err := FindDataset(args[0])
		ExitIfErr(err)

		file, err := dsfs.LoadDatasetData(store, ds)
		ExitIfErr(err)

		os.Open(path)

		ioutil.WriteFile(o, data, perm)
	},
}

func init() {
	RootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("output", "o", "dataset", "path to write to")
	exportCmd.Flags().BoolP("data-only", "d", false, "write data only (no package)")
	// exportCmd.Flags().BoolP("zip", "z", false, "compress export as zip archive")
	// exportCmd.Flags().StringP("format", "f", "csv", "set output format [csv,json]")
}
