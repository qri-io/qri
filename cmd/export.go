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
	"encoding/json"
	"fmt"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "export dataset data",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("please specify a dataset name to export")
			return
		}
		path := cmd.Flag("output").Value.String()
		if path == "" {
			// TODO - support printing to stdout
			ErrExit(fmt.Errorf("please specify an output path"))
		}

		r := GetRepo()
		store, err := GetIpfsFilestore()
		ExitIfErr(err)

		ds, err := FindDataset(r, store, args[0])
		ExitIfErr(err)

		if cmd.Flag("data-only").Value.String() == "true" {
			src, err := dsfs.LoadDatasetData(store, ds)
			ExitIfErr(err)

			dst, err := os.Create(fmt.Sprintf("%s.%s", path, ds.Structure.Format.String()))
			ExitIfErr(err)

			_, err = io.Copy(dst, src)
			ExitIfErr(err)

			err = dst.Close()
			ExitIfErr(err)
			return
		}

		if cmd.Flag("zip").Value.String() == "true" {
			dst, err := os.Create(fmt.Sprintf("%s.zip", path))
			ExitIfErr(err)

			err = writePackage(store, ds, dst)
			ExitIfErr(err)
			err = dst.Close()
			ExitIfErr(err)
			return
		}

		writeDir(store, ds, path)
	},
}

// TODO - move this somewhere more useful, like that defunct dataset subpackage
func writePackage(store cafs.Filestore, ds *dataset.Dataset, w io.Writer) error {
	zw := zip.NewWriter(w)

	dsf, err := zw.Create(dsfs.PackageFileDataset.String())
	if err != nil {
		return err
	}
	dsdata, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}
	_, err = dsf.Write(dsdata)
	if err != nil {
		return err
	}

	datadst, err := zw.Create(fmt.Sprintf("data.%s", ds.Structure.Format.String()))
	if err != nil {
		return err
	}

	datasrc, err := dsfs.LoadDatasetData(store, ds)
	if err != nil {
		return err
	}

	if _, err = io.Copy(datadst, datasrc); err != nil {
		return err
	}

	return zw.Close()
}

func writeDir(store cafs.Filestore, ds *dataset.Dataset, path string) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	dsdata, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(path, dsfs.PackageFileDataset.String()), dsdata, os.ModePerm)
	if err != nil {
		return err
	}

	datasrc, err := dsfs.LoadDatasetData(store, ds)
	if err != nil {
		return err
	}

	datadst, err := os.Create(filepath.Join(path, fmt.Sprintf("data.%s", ds.Structure.Format.String())))
	if err != nil {
		return err
	}
	if _, err = io.Copy(datadst, datasrc); err != nil {
		return err
	}

	return nil
}

func init() {
	RootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("output", "o", "dataset", "path to write to")
	exportCmd.Flags().BoolP("data-only", "d", false, "write data only (no package)")
	exportCmd.Flags().BoolP("zip", "z", false, "compress export as zip archive")
	// exportCmd.Flags().StringP("format", "f", "csv", "set output format [csv,json]")
}
