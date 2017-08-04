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
	"os"
	"path/filepath"

	"github.com/ipfs/go-datastore"
	ipfs "github.com/qri-io/castore/ipfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"

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
					if r := InferResource(fi); r != nil {
						foundFiles[fi.Name()] = r
						_, err = AddFileResource(ds, ns, r, filepath.Join(base, fi.Name()), true)
						ExitIfErr(err)
					}
				}
			}
		} else {
			file, err := os.Stat(base)
			ExitIfErr(err)

			if r := InferResource(file); r != nil {
				_, err = AddFileResource(ds, ns, r, base, passive)
				ExitIfErr(err)
			}
			// PrintSuccess("successfully initialized dataset %s: %s")
			// PrintDatasetDetailedInfo(ds)
		}

		err = SaveNamespaceGraph(ns)
		ExitIfErr(err)
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&rescursive, "recursive", "r", false, "recursive add from a directory")
	initCmd.Flags().BoolVarP(&passive, "passive", "p", false, "disable interactive init")
}

func InferResource(file os.FileInfo) *dataset.Resource {
	// only work with files that have a proper extension
	if _, err := detect.ExtensionDataFormat(file.Name()); err != nil {
		return nil
	}

	if r, err := detect.FromFile(file.Name()); err == nil {
		return r
	} else {
		PrintWarning("error with file '%s': %s", file.Name(), err.Error())
		return nil
	}
	return nil
}

func AddFileResource(ds *ipfs.Datastore, ns map[string]datastore.Key, r *dataset.Resource, path string, passive bool) (rkey datastore.Key, err error) {
	rkey = datastore.NewKey("")
	var adr string
	// TODO - extract a default name from the file name
	// TODO - require this be a proper, no-space alphanumeric type thing
	if !passive {
		adr = InputText(fmt.Sprintf("choose a variable name for %s", path), path)
		if err != nil {
			return
		}
	} else {
		adr = path
	}

	datahash, err := ds.AddAndPinPath(path)
	r.Path = datastore.NewKey("/ipfs/" + datahash)

	rdata, err := r.MarshalJSON()
	if err != nil {
		return
	}
	rhash, err := ds.AddAndPinBytes(rdata)
	if err != nil {
		return
	}

	rkey = datastore.NewKey("/ipfs/" + rhash)
	ns[adr] = rkey

	return
}
