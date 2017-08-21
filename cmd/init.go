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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	ipfs "github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/qri/core/datasets"

	"github.com/spf13/cobra"
)

var passive bool
var rescursive bool

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a dataset, adding it to your local collection of datasets",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var adr string

		base := args[0]
		ns := LoadNamespaceGraph()
		ds, err := ipfs.NewDatastore()
		ExitIfErr(err)

		if rescursive {
			files, err := ioutil.ReadDir(base)
			ExitIfErr(err)
			foundFiles := map[string]*dataset.Resource{}
			for _, fi := range files {
				if fi.IsDir() {
					continue
				} else {
					adr = fi.Name()
					rsc, err := detect.FromFile(adr)
					ExitIfErr(err)
					// Add to the namespace as the filename
					// TODO - require this be a proper, no-space alphanumeric type thing
					foundFiles[adr] = rsc

					rkey, err := datasets.AddFileResource(ds, filepath.Join(base, fi.Name()), rsc)
					ExitIfErr(err)
					ns[adr] = rkey
				}
			}
		} else {
			file, err := os.Stat(base)
			ExitIfErr(err)

			// TODO - extract a default name from the file name
			// TODO - require this be a proper, no-space alphanumeric type thing
			if !passive {
				adr = InputText(fmt.Sprintf("choose a variable name for %s", file.Name()), file.Name())
				if err != nil {
					return
				}
			} else {
				adr = file.Name()
			}

			rsc, err := detect.FromFile(file.Name())
			ExitIfErr(err)

			rkey, err := datasets.AddFileResource(ds, base, rsc)
			ExitIfErr(err)

			// Add to the namespace as the filename
			// TODO - require this be a proper, no-space alphanumeric type thing
			ns[adr] = rkey
			// PrintSuccess("successfully initialized dataset %s: %s")
			// PrintDatasetDetailedInfo(ds)
		}

		err = SaveNamespaceGraph(ns)
		ExitIfErr(err)
	},
}

func init() {
	flag.Parse()
	RootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&rescursive, "recursive", "r", false, "recursive add from a directory")
	initCmd.Flags().BoolVarP(&passive, "passive", "p", false, "disable interactive init")
}
