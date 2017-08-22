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
	"github.com/ipfs/go-datastore"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	ipfs "github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"

	"github.com/spf13/cobra"
)

var (
	initFile       string
	initName       string
	initPassive    bool
	initRescursive bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a dataset, adding it to your local collection of datasets",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if initFile == "" {
			ErrExit(fmt.Errorf("please provide a file argument"))
		}

		path, err := filepath.Abs(initFile)
		ExitIfErr(err)
		fmt.Println(path)

		ns := LoadNamespaceGraph()
		ds, err := ipfs.NewDatastore()
		ExitIfErr(err)

		if initRescursive {
			files, err := ioutil.ReadDir(path)
			ExitIfErr(err)
			foundFiles := map[string]datastore.Key{}
			for _, fi := range files {
				if fi.IsDir() {
					continue
				} else {
					initName = fi.Name()
					st, err := detect.FromFile(initName)
					ExitIfErr(err)
					// Add to the namespace as the filename
					// TODO - require this be a proper, no-space alphanumeric type thing

					datahash, err := ds.AddAndPinPath(filepath.Join(path, fi.Name()))
					ExitIfErr(err)
					datakey := datastore.NewKey("/ipfs/" + datahash)

					// rkey, dskey, err := datasets.AddFileStructure(ds, filepath.Join(path, fi.Name()), rsc)
					d := &dataset.Dataset{
						Timestamp: time.Now().In(time.UTC),
						Structure: st,
						Data:      datakey,
					}

					dspath, err := d.Save(ds)
					ExitIfErr(err)

					foundFiles[initName] = dspath
					ns[initName] = dspath
				}
			}
		} else {
			file, err := os.Stat(path)
			ExitIfErr(err)

			// TODO - extract a default name from the file name
			// TODO - require this be a proper, no-space alphanumeric type thing
			if !initPassive && initName == "" {
				initName = InputText(fmt.Sprintf("choose a variable name for %s", file.Name()), file.Name())
				if err != nil {
					return
				}
			} else if initName == "" {
				initName = file.Name()
			}

			st, err := detect.FromFile(path)
			ExitIfErr(err)

			datahash, err := ds.AddAndPinPath(path)
			ExitIfErr(err)
			datakey := datastore.NewKey("/ipfs/" + datahash)

			d := &dataset.Dataset{
				Timestamp: time.Now().In(time.UTC),
				Structure: st,
				Data:      datakey,
			}

			dspath, err := d.Save(ds)
			ExitIfErr(err)

			// Add to the namespace as the filename
			// TODO - require this be a proper, no-space alphanumeric type thing
			ns[initName] = dspath

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
	initCmd.Flags().StringVarP(&initFile, "file", "f", "", "data file to initialize from")
	initCmd.Flags().StringVarP(&initName, "name", "n", "", "name to give dataset")
	initCmd.Flags().BoolVarP(&initRescursive, "recursive", "r", false, "recursive add from a directory")
	initCmd.Flags().BoolVarP(&initPassive, "passive", "p", false, "disable interactive init")
}
