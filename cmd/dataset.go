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
	"os"

	"github.com/spf13/cobra"
)

// datasetCmd represents the dataset command
var datasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Dataset manipulation methods",
	Long: `Datasets are logical containers of data on qri.

	This is a big experiment, so, uh, more documentation to follow.`,
}

func init() {

	datasetCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "get the status of the dataset in the current working directory",
		Long:  "TODO",
		Run: func(cmd *cobra.Command, args []string) {
			repo := &Repo{}
			dir := GetWd()

			if err := repo.ReadLocal(dir); err != nil {
				fmt.Printf("Error getting working directory: %s", err.Error())
				os.Exit(1)
			}
		},
	})

	datasetCmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "initialize a new qri repository",
		Long:  "TODO",
		Run: func(cmd *cobra.Command, args []string) {
			repo := &Repo{}
			dir := GetWd()

			if err := repo.Init(dir); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	})

	datasetCmd.AddCommand(&cobra.Command{
		Use:   "add",
		Short: "add current files",
		Long:  "TODO",
		Run: func(cmd *cobra.Command, args []string) {
			repo := &Repo{}
			dir := GetWd()

			if err := repo.Add(dir); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	})

	datasetCmd.AddCommand(&cobra.Command{
		Use:   "commit",
		Short: "TODO",
		Long:  "TODO",
		Run: func(cmd *cobra.Command, args []string) {
			repo := &Repo{}
			dir := GetWd()

			if err := repo.Commit(dir); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	})

	datasetCmd.AddCommand(&cobra.Command{
		Use:   "push",
		Short: "TODO",
		Long:  "TODO",
		Run: func(cmd *cobra.Command, args []string) {
			repo := &Repo{}
			dir := GetWd()

			if err := repo.Push(dir); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	})

	datasetCmd.AddCommand(&cobra.Command{
		Use:   "pull",
		Short: "TODO",
		Long:  "TODO",
		Run: func(cmd *cobra.Command, args []string) {
			repo := &Repo{}
			dir := GetWd()

			if err := repo.Pull(dir); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	})

	RootCmd.AddCommand(datasetCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// datasetCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// datasetCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
