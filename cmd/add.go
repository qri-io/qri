package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	addDsFilepath     string
	addDsMetaFilepath string
	addDsName         string
	addDsURL          string
	addDsPassive      bool
)

var datasetAddCmd = &cobra.Command{
	Use:        "add",
	Short:      "add a dataset to your local repository",
	SuggestFor: []string{"init"},
	Long: `
Usage:
	qri add [--meta <file>] [--structure <file>] <data file>

Add creates a new dataset from data you supply. You can supply data from a file 
or a URL. Please note that all data added to qri is made public on the 
distributed web when you run qri connect.

When adding data, you can supply metadata and dataset structure, but it’s not 
required. qri does what it can to infer the details you don’t provide. 
add currently supports two data formats:
- CSV (Comma Separated Values)
- JSON (Javascript Object Notation)

Once you’ve added data, you can use the export command to pull the data out of 
qri, change the data outside of qri, and use the save command to record those 
changes to qri`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			if !strings.HasSuffix(args[0], dsfs.PackageFileDataset.String()) {
				ErrExit(fmt.Errorf("invalid dataset path. paths should be /ipfs/[hash]/dataset.json"))
			}

			if addDsName == "" {
				ErrExit(fmt.Errorf("please provide a --name"))
			}

			req, err := datasetRequests(false)
			ExitIfErr(err)

			root := strings.TrimSuffix(args[0], "/"+dsfs.PackageFileDataset.String())
			p := &core.AddParams{
				Name: addDsName,
				Hash: root,
			}
			res := &repo.DatasetRef{}
			err = req.Add(p, res)
			ExitIfErr(err)

			printInfo("Successfully added dataset %s: %s", addDsName, res.Path.String())
		} else {
			initDataset()
		}
	},
}

func initDataset() {
	var (
		dataFile, metaFile *os.File
		err                error
	)

	if addDsFilepath == "" && addDsURL == "" {
		ErrExit(fmt.Errorf("please provide either a file or a url argument"))
	} else if addDsName == "" {
		ErrExit(fmt.Errorf("please provide a --name"))
	}

	dataFile, err = loadFileIfPath(addDsFilepath)
	ExitIfErr(err)
	metaFile, err = loadFileIfPath(addDsMetaFilepath)
	ExitIfErr(err)

	req, err := datasetRequests(false)
	ExitIfErr(err)

	p := &core.InitParams{
		Name:         addDsName,
		URL:          addDsURL,
		DataFilename: filepath.Base(addDsFilepath),
	}

	// this is because passing nil to interfaces is bad
	// see: https://golang.org/doc/faq#nil_error
	if dataFile != nil {
		p.Data = dataFile
	}
	if metaFile != nil {
		p.Metadata = metaFile
	}

	ref := &repo.DatasetRef{}
	err = req.Init(p, ref)
	ExitIfErr(err)
	// req.Get(&core.GetDatasetParams{ Name: p.Name }, res)
	printSuccess("initialized dataset %s: %s", ref.Name, ref.Path.String())
}

func init() {
	datasetAddCmd.Flags().StringVarP(&addDsName, "name", "n", "", "name to give dataset")
	datasetAddCmd.Flags().StringVarP(&addDsURL, "url", "u", "", "url to file to initialize from")
	datasetAddCmd.Flags().StringVarP(&addDsFilepath, "file", "f", "", "data file to initialize from")
	datasetAddCmd.Flags().StringVarP(&addDsMetaFilepath, "meta", "m", "", "dataset metadata file")
	datasetAddCmd.Flags().BoolVarP(&addDsPassive, "passive", "p", false, "disable interactive init")
	RootCmd.AddCommand(datasetAddCmd)
}
