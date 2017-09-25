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
	"github.com/qri-io/dataset/dsfs"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	initFile       string
	initMetaFile   string
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

		r := GetRepo()
		// ns := LoadNamespaceGraph()
		ds, err := GetIpfsFilestore()
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

					datahash, err := ds.AddPath(filepath.Join(path, fi.Name()), true)
					ExitIfErr(err)
					datakey := datastore.NewKey("/ipfs/" + datahash)

					// rkey, dskey, err := datasets.AddFileStructure(ds, filepath.Join(path, fi.Name()), rsc)
					d := &dataset.Dataset{
						Timestamp: time.Now().In(time.UTC),
						Structure: st,
						Data:      datakey,
					}

					dspath, err := dsfs.SaveDataset(ds, d, true)
					ExitIfErr(err)

					foundFiles[initName] = dspath
					r.PutName(initName, dspath)
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
				initName = repo.CoerceDatasetName(file.Name())
			}

			if !repo.ValidDatasetName(initName) {
				ErrExit(fmt.Errorf("invalid dataset name: %s", initName))
			}

			st, err := detect.FromFile(path)
			ExitIfErr(err)

			datahash, err := ds.AddPath(path, true)
			ExitIfErr(err)
			datakey := datastore.NewKey("/ipfs/" + datahash)

			d := &dataset.Dataset{}

			// parse any provided metadata
			if initMetaFile != "" {
				mdata, err := ioutil.ReadFile(initMetaFile)
				if err != nil {
					ErrExit(fmt.Errorf("error opening metadata file: %s", err.Error()))
				}
				if err := d.UnmarshalJSON(mdata); err != nil {
					ErrExit(fmt.Errorf("error parsing metadata file: %s", err.Error()))
				}
			}

			if d.Structure == nil {
				d.Structure = &dataset.Structure{}
			}

			// structure may have been set by the metadata file above
			// by calling assign on ourselves with inferred structure in
			// the middle, any user-contributed schema metadata will overwrite
			// inferred metadata, but inferred schema properties will populate
			// empty fields
			d.Structure.Assign(st, d.Structure)
			d.Timestamp = time.Now().In(time.UTC)
			d.Data = datakey
			d.Length = int(file.Size())

			dspath, err := dsfs.SaveDataset(ds, d, true)
			ExitIfErr(err)

			// Add to the namespace as the filename
			// TODO - require this be a proper, no-space alphanumeric type thing
			// ns[initName] = dspath
			err = r.PutName(initName, dspath)
			ExitIfErr(err)

			PrintSuccess("initialized dataset %s: %s", initName, dspath)
			// PrintDatasetDetailedInfo(ds)
		}
	},
}

func init() {
	flag.Parse()
	RootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initFile, "file", "f", "", "data file to initialize from")
	initCmd.Flags().StringVarP(&initName, "name", "n", "", "name to give dataset")
	initCmd.Flags().StringVarP(&initMetaFile, "meta", "m", "", "dataset metadata")
	initCmd.Flags().BoolVarP(&initRescursive, "recursive", "r", false, "recursive add from a directory")
	initCmd.Flags().BoolVarP(&initPassive, "passive", "p", false, "disable interactive init")
}
