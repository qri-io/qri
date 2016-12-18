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
	"io/ioutil"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset_detect"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a dataset",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		base := GetWd()

		files, err := ioutil.ReadDir(base)
		if err != nil {
			ErrExit(err)
		}

		dataset := &dataset.Dataset{}
		foundFiles := map[string][]byte{}
		for _, fi := range files {
			if fi.IsDir() {
				continue
			} else if ds, err := dataset_detect.FromFile(fi.Name()); err == nil {
				foundFiles[fi.Name()] = ds.Data
				ds.Data = nil

				dataset.Datasets = append(dataset.Datasets, ds)

				break
			}
		}

		if len(dataset.Datasets) == 1 {
			dataset = dataset.Datasets[0]
			dataset.Datasets = nil
		} else if len(dataset.Datasets) == 0 {
			dataset = nil
		}

		adr, err := InputAddress("", dataset.Address)
		ExitIfErr(err)
		dataset.Address = adr

		// if err := history.Init(store, func(o *history.InitOpt) {
		// 	o.Dataset = dataset
		// }); err != nil {
		// 	ErrExit(err)
		// }
		// fmt.Printf("created new repository at %s\n", base)

		err = WriteDataset(Cache(), dataset, foundFiles)
		ExitIfErr(err)
		PrintSuccess("successfully initialized dataset at: %s%s", cachePath(), DatasetPath(dataset))
		PrintDatasetDetailedInfo(dataset)
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
	// TODO - add passive flag to disable interacive input.
}
