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
	ipfs "github.com/qri-io/castore/ipfs"
	"github.com/spf13/cobra"
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
		ns := LoadNamespaceGraph()
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
		ds, err := ipfs.NewDatastore()
		ExitIfErr(err)

		ns := LoadNamespaceGraph()

		path := datastore.NewKey(args[0])
		if ns[args[0]].String() != "" {
			path = ns[args[0]]
		}

		resource, err := GetResource(ds, path)
		ExitIfErr(err)

		out, err := json.MarshalIndent(resource, "", "  ")
		ExitIfErr(err)

		fmt.Printf("%s\n", string(out))
	},
}

var datasetAddCmd = &cobra.Command{
	Use:   "add",
	Short: "add a dataset to your local namespace based on a resource hash",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			ErrExit(fmt.Errorf("wrong number of arguments for adding a dataset, expected [name] [resource hash]"))
		}
		ns := LoadNamespaceGraph()
		ns[args[0]] = datastore.NewKey(args[1])
		err := SaveNamespaceGraph(ns)
		ExitIfErr(err)
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
		ns := LoadNamespaceGraph()
		delete(ns, args[0])
		err := SaveNamespaceGraph(ns)
		ExitIfErr(err)
	},
}

func init() {
	datasetCmd.AddCommand(datasetListCmd)
	datasetCmd.AddCommand(datasetInfoCmd)
	datasetCmd.AddCommand(datasetAddCmd)
	datasetCmd.AddCommand(datasetRemoveCmd)
	RootCmd.AddCommand(datasetCmd)
}
