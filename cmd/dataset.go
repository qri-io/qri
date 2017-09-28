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
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/spf13/cobra"
	"strings"
)

// datasetCmd represents the dataset commands
var datasetCmd = &cobra.Command{
	Use:     "dataset",
	Aliases: []string{"ds"},
	Short:   "dataset tools",
	Long:    ``,
}

var datasetListCmd = &cobra.Command{
	Use:   "list",
	Short: "list your local datasets",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ns, err := GetRepo().Namespace(100, 0)
		ExitIfErr(err)
		for name, resource := range ns {
			PrintInfo("%s\t\t: %s", name, resource.String())
		}
	},
}

var datasetInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "get information about a dataset",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			ErrExit(fmt.Errorf("wrong number of arguments. expected qri info [dataset_name]"))
		}
		ds, err := GetIpfsFilestore()
		ExitIfErr(err)

		path, err := GetRepo().GetPath(args[0])
		ExitIfErr(err)

		d, err := dsfs.LoadDataset(ds, path)
		ExitIfErr(err)

		out, err := json.MarshalIndent(d, "", "  ")
		ExitIfErr(err)

		fmt.Printf("%s\n", string(out))
	},
}

var datasetAddCmd = &cobra.Command{
	Use:   "add",
	Short: "add a dataset to your local namespace based on a resource hash",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			ErrExit(fmt.Errorf("wrong number of arguments for adding a dataset, expected qri dataset add [resource hash]"))
		}
		if !strings.HasSuffix(args[0], dsfs.PackageFileDataset.String()) {
			ErrExit(fmt.Errorf("invalid dataset path. paths should be /ipfs/[hash]/dataset.json"))
		}

		r := GetRepo()
		fs, err := GetIpfsFilestore()
		ExitIfErr(err)

		name := cmd.Flag("name").Value.String()
		if name == "" {
			ErrExit(fmt.Errorf("please provide a name for the dataset using the --name flag"))
		}

		root := strings.TrimSuffix(args[0], "/"+dsfs.PackageFileDataset.String())

		PrintInfo("downloading %s...", root)
		err = fs.Pin(datastore.NewKey(root), true)
		ExitIfErr(err)

		err = r.PutName(name, datastore.NewKey(args[0]))
		ExitIfErr(err)

		PrintInfo("Successfully added dataset %s : %s", name, args[0])
	},
}

var datasetRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "remove a dataset from your local namespace based on a resource hash",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			ErrExit(fmt.Errorf("wrong number of arguments for adding a dataset, expected [name]"))
		}
		name := args[0]

		fs, err := GetIpfsFilestore()
		ExitIfErr(err)

		r := GetRepo()
		path, err := r.GetPath(name)
		ExitIfErr(err)

		root := datastore.NewKey(strings.TrimSuffix(path.String(), "/"+dsfs.PackageFileDataset.String()))

		err = fs.Delete(root)
		ExitIfErr(err)

		err = r.DeleteDataset(path)
		ExitIfErr(err)

		r.DeleteName(name)
		ExitIfErr(err)

		PrintSuccess("removed dataset %s: %s", name, path)
	},
}

func init() {
	datasetCmd.AddCommand(datasetListCmd)
	datasetCmd.AddCommand(datasetInfoCmd)
	datasetCmd.AddCommand(datasetAddCmd)
	datasetCmd.AddCommand(datasetRemoveCmd)

	datasetAddCmd.Flags().StringP("name", "n", "", "local name for dataset")

	RootCmd.AddCommand(datasetCmd)
}
