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
	"os"
	"path/filepath"

	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	initFile     string
	initMetaFile string
	initName     string
	initUrl      string
	initPassive  bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a dataset, adding it to your local collection of datasets",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var dataFile, metaFile *os.File

		if initFile == "" && initUrl == "" {
			ErrExit(fmt.Errorf("please provide either a file or a url argument"))
		}

		if initName == "" {
			ErrExit(fmt.Errorf("please provide a --name"))
		}

		if initFile != "" {
			filepath, err := filepath.Abs(initFile)
			ExitIfErr(err)
			dataFile, err = os.Open(filepath)
			ExitIfErr(err)
		}

		if initMetaFile != "" {
			filepath, err := filepath.Abs(initMetaFile)
			ExitIfErr(err)
			metaFile, err = os.Open(filepath)
			ExitIfErr(err)
		}

		r := GetRepo(false)
		store, err := GetIpfsFilestore(false)
		ExitIfErr(err)
		req := core.NewDatasetRequests(store, r)

		p := &core.InitDatasetParams{
			Name:         initName,
			Url:          initUrl,
			DataFilename: filepath.Base(initFile),
		}

		// this is because passing nil to interfaces is bad: https://golang.org/doc/faq#nil_error
		if dataFile != nil {
			p.Data = dataFile
		}
		if metaFile != nil {
			p.Metadata = metaFile
		}

		ref := &repo.DatasetRef{}
		err = req.InitDataset(p, ref)
		ExitIfErr(err)
		// req.Get(&core.GetDatasetParams{ Name: p.Name }, res)
		PrintSuccess("initialized dataset %s: %s", ref.Name, ref.Path.String())
	},
}

func init() {
	flag.Parse()
	RootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initUrl, "url", "u", "", "url to file to initialize from")
	initCmd.Flags().StringVarP(&initFile, "file", "f", "", "data file to initialize from")
	initCmd.Flags().StringVarP(&initName, "name", "n", "", "name to give dataset")
	initCmd.Flags().StringVarP(&initMetaFile, "meta", "m", "", "dataset metadata")
	initCmd.Flags().BoolVarP(&initPassive, "passive", "p", false, "disable interactive init")
}
