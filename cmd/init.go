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
	"io/ioutil"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset_detect"

	"github.com/qri-io/history"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a qri repository",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		base := GetWd()
		store := Store(cmd, args)

		files, err := ioutil.ReadDir(base)
		if err != nil {
			ErrExit(err)
		}

		dataset := &dataset.Dataset{}
		for _, fi := range files {
			if fi.IsDir() {
				continue
			} else if ds, err := dataset_detect.FromFile(fi.Name()); err == nil {
				ds.Data = nil
				dataset.Datasets = append(dataset.Datasets, ds)
			}
		}

		if len(dataset.Datasets) == 1 {
			dataset = dataset.Datasets[0]
			dataset.Datasets = nil
		} else if len(dataset.Datasets) == 0 {
			dataset = nil
		}

		if err := history.Init(store, func(o *history.InitOpt) {
			o.Dataset = dataset
		}); err != nil {
			ErrExit(err)
		}

		fmt.Printf("created new repository at %s\n", base)
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
}
