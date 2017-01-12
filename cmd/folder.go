// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
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

	"github.com/qri-io/dataset"

	"github.com/qri-io/namespace"
	"github.com/qri-io/namespace/local"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// foldersCmd represents the folders command
var folderCmd = &cobra.Command{
	Use:   "folder",
	Short: "List & edit folders",
	// Long:  `Namespaces are a domain connected with a base address.`,
}

var folderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List folders",
	Run: func(cmd *cobra.Command, args []string) {
		for i, r := range GetFolders(cmd, args) {
			PrintInfo("%d. %s", i+1, r.Path)
		}
	},
}

var folderAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a folder",
	Run: func(cmd *cobra.Command, args []string) {
		PrintNotYetFinished(cmd)
	},
}

var folderRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a folder",
	Run: func(cmd *cobra.Command, args []string) {
		PrintNotYetFinished(cmd)
	},
}

func init() {
	folderCmd.AddCommand(folderListCmd)
	folderCmd.AddCommand(folderAddCmd)
	folderCmd.AddCommand(folderRemoveCmd)
	RootCmd.AddCommand(folderCmd)
}

type Folder struct {
	Path    string          `json:"path"`
	Address dataset.Address `json:"address"`
}

func (f *Folder) Namespace() namespace.Namespace {
	return local.NewNamespaceFromPath(f.Path)
}

func GetFolders(cmd *cobra.Command, args []string) []*Folder {
	foldersList := viper.Get("folders")
	if folderSlice, ok := foldersList.([]interface{}); ok {
		folders := []*Folder{}
		for _, nsI := range folderSlice {
			if ns, ok := nsI.(map[string]interface{}); ok {
				path := iFaceStr(ns["path"])
				adr := iFaceStr(ns["address"])
				folders = append(folders, &Folder{Path: path, Address: dataset.NewAddress(adr)})
			} else {
				ErrExit(fmt.Errorf("invalid folders configuration. Check your config file!"))
			}
		}

		// add working dir
		folders = append(folders, &Folder{Path: GetWd(), Address: dataset.NewAddress()})

		return folders
	} else {
		ErrExit(fmt.Errorf("invalid folders configuration. Check your config file!"))
	}
	return nil
}

func LocalNamespaces(cmd *cobra.Command, args []string) (ns Namespaces) {
	for _, f := range GetFolders(cmd, args) {
		ns = append(ns, f.Namespace())
	}
	return
}
