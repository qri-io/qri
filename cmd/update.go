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
	"github.com/spf13/cobra"
)

var (
	updateFile       string
	updateMetaFile   string
	updateName       string
	updatePassive    bool
	updateRescursive bool
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a dataset, changing metadata and/or data",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		r := GetRepo(false)
		store, err := GetIpfsFilestore(false)
		ExitIfErr(err)

		ref, err := DatasetRef(r, store, args[0])
		ExitIfErr(err)

		var datapath string
		if updateFile != "" {
			datapath, err = filepath.Abs(updateFile)
			ExitIfErr(err)
		}

		if updateMetaFile == "" && datapath == "" {
			ErrExit(fmt.Errorf("either a metadata or data option is required"))
		}

		if updateRescursive {
			// TODO - this is currently unreachable & unsinished
			// files, err := ioutil.ReadDir(path)
			// ExitIfErr(err)
			// foundFiles := map[string]datastore.Key{}
			// for _, fi := range files {
			// 	if fi.IsDir() {
			// 		continue
			// 	} else {
			// 		updateName = fi.Name()
			// 		st, err := detect.FromFile(updateName)
			// 		ExitIfErr(err)
			// 		// Add to the namespace as the filename
			// 		// TODO - require this be a proper, no-space alphanumeric type thing

			// 		datahash, err := ds.AddPath(filepath.Join(path, fi.Name()), true)
			// 		ExitIfErr(err)
			// 		datakey := datastore.NewKey("/ipfs/" + datahash)

			// 		// rkey, dskey, err := datasets.AddFileStructure(ds, filepath.Join(path, fi.Name()), rsc)
			// 		d := &dataset.Dataset{
			// 			Timestamp: time.Now().In(time.UTC),
			// 			Structure: st,
			// 			Data:      datakey,
			// 		}

			// 		dspath, err := dsfs.SaveDataset(ds, d, true)
			// 		ExitIfErr(err)

			// 		foundFiles[updateName] = dspath
			// 		r.PutName(updateName, dspath)
			// 	}
			// }
		} else {
			ds := &dataset.Dataset{}

			// add all previous fields
			ds.Assign(ref.Dataset)

			// parse any provided metadata
			if updateMetaFile != "" {
				updates := &dataset.Dataset{}
				mdata, err := ioutil.ReadFile(updateMetaFile)
				if err != nil {
					ErrExit(fmt.Errorf("error opening metadata file: %s", err.Error()))
				}
				if err := updates.UnmarshalJSON(mdata); err != nil {
					ErrExit(fmt.Errorf("error parsing metadata file: %s", err.Error()))
				}

				ds.Assign(updates)
			}

			if datapath != "" {
				file, err := os.Stat(datapath)
				ExitIfErr(err)

				datahash, err := store.AddPath(datapath, true)
				ExitIfErr(err)
				ds.Data = datastore.NewKey("/ipfs/" + datahash)
				ds.Length = int(file.Size())
			}

			// TODO - validate dataset structure
			// structure may have been set by the metadata file above
			// by calling assign on ourselves with inferred structure in
			// the middle, any user-contributed schema metadata will overwrite
			// inferred metadata, but inferred schema properties will populate
			// empty fields
			// ds.Structure.Assign(ds.Structure, d.Structure)

			ds.Timestamp = time.Now().In(time.UTC)
			ds.Previous = ref.Path

			dspath, err := dsfs.SaveDataset(store, ds, true)
			ExitIfErr(err)

			err = r.PutName(updateName, dspath)
			ExitIfErr(err)

			PrintSuccess("updated dataset %s: %s", updateName, dspath)
			// PrintDatasetDetailedInfo(ds)
		}
	},
}

func init() {
	flag.Parse()
	RootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "data file to updateialize from")
	updateCmd.Flags().StringVarP(&updateMetaFile, "meta", "m", "", "dataset metadata updates")
	// updateCmd.Flags().StringVarP(&updateName, "name", "n", "", "name to give dataset")
	// TODO - one day...
	// updateCmd.Flags().BoolVarP(&updateRescursive, "recursive", "r", false, "recursive add from a directory")
	// updateCmd.Flags().BoolVarP(&updatePassive, "passive", "p", false, "disable interactive update")
}
