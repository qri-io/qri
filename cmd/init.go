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

	"github.com/qri-io/repo"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		base := GetWd()
		r := repo.NewRepo(func(o *repo.RepoOpt) {
			o.BasePath = base
		})

		files, err := ioutil.ReadDir(base)
		if err != nil {
			ErrExit(err)
		}

		dataset := &dataset.Dataset{}
		for _, fi := range files {
			if fi.IsDir() {
				continue
			} else if ds, err := dataset_detect.FromFile(fi.Name()); err == nil {
				dataset.Datasets = append(dataset.Datasets, ds)
			}
		}

		if len(dataset.Datasets) == 1 {
			dataset = dataset.Datasets[0]
			dataset.Datasets = nil
		} else if len(dataset.Datasets) == 0 {
			dataset = nil
		}

		if err := r.Init(func(o *repo.InitOpt) {
			o.Dataset = dataset
		}); err != nil {
			ErrExit(err)
		}

		fmt.Printf("created new repository at %s\n", base)
	},
}

func init() {
	RootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
