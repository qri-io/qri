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
	"archive/zip"
	"io/ioutil"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/fs"
	"github.com/qri-io/namespace"
	"github.com/qri-io/namespace/local"
	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download dataset(s)",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		namespaces := GetNamespaces(cmd, args)
		for _, ns := range namespaces {
			// ignore local namespace
			if _, ok := ns.Namespace.(*local.Namespace); ok {
				continue
			}

			adr := GetAddress(cmd, args)
			_, err := downloadPackage(ns, adr)
			ExitIfErr(err)
			PrintSuccess("downloaded %s to %s", adr.String(), adr.PathString())
		}
	},
}

func init() {
	RootCmd.AddCommand(downloadCmd)
}

func downloadPackage(ns namespace.Namespace, adr dataset.Address) (fs.Store, error) {
	store := Cache()
	r, size, err := ns.Package(adr)
	if err != nil {
		return store, err
	}

	buf := make([]byte, size)
	if _, err := r.ReadAt(buf, 0); err != nil {
		return store, err
	}

	zipr, err := zip.NewReader(r, size)
	if err != nil {
		return store, err
	}

	for _, f := range zipr.File {
		r, err := f.Open()
		if err != nil {
			return store, err
		}

		data, err := ioutil.ReadAll(r)
		if err != nil {
			return store, err
		}

		if err := store.Write(filepath.Join(adr.PathString(), f.Name), data); err != nil {
			return store, err
		}
	}

	return store, nil
}
