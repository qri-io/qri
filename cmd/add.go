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
	addDsUrl          string
	addDsPassive      bool
)

var datasetAddCmd = &cobra.Command{
	Use:   "add",
	Short: "add a dataset",
	Long:  `add a dataset to your local namespace based on a resource hash, local file, or url`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			if !strings.HasSuffix(args[0], dsfs.PackageFileDataset.String()) {
				ErrExit(fmt.Errorf("invalid dataset path. paths should be /ipfs/[hash]/dataset.json"))
			}

			if addDsName == "" {
				ErrExit(fmt.Errorf("please provide a --name"))
			}

			req, err := DatasetRequests(false)
			ExitIfErr(err)

			root := strings.TrimSuffix(args[0], "/"+dsfs.PackageFileDataset.String())
			p := &core.AddParams{
				Name: addDsName,
				Hash: root,
			}
			res := &repo.DatasetRef{}
			err = req.AddDataset(p, res)
			ExitIfErr(err)

			PrintInfo("Successfully added dataset %s: %s", addDsName, res.Path.String())
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

	if addDsFilepath == "" && addDsUrl == "" {
		ErrExit(fmt.Errorf("please provide either a file or a url argument"))
	} else if addDsName == "" {
		ErrExit(fmt.Errorf("please provide a --name"))
	}

	dataFile, err = loadFileIfPath(addDsFilepath)
	ExitIfErr(err)
	metaFile, err = loadFileIfPath(addDsMetaFilepath)
	ExitIfErr(err)

	req, err := DatasetRequests(false)
	ExitIfErr(err)

	p := &core.InitDatasetParams{
		Name:         addDsName,
		Url:          addDsUrl,
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
	err = req.InitDataset(p, ref)
	ExitIfErr(err)
	// req.Get(&core.GetDatasetParams{ Name: p.Name }, res)
	PrintSuccess("initialized dataset %s: %s", ref.Name, ref.Path.String())
}

func init() {
	datasetAddCmd.Flags().StringVarP(&addDsName, "name", "n", "", "name to give dataset")
	datasetAddCmd.Flags().StringVarP(&addDsUrl, "url", "u", "", "url to file to initialize from")
	datasetAddCmd.Flags().StringVarP(&addDsFilepath, "file", "f", "", "data file to initialize from")
	datasetAddCmd.Flags().StringVarP(&addDsMetaFilepath, "meta", "m", "", "dataset metadata file")
	datasetAddCmd.Flags().BoolVarP(&addDsPassive, "passive", "p", false, "disable interactive init")
	RootCmd.AddCommand(datasetAddCmd)
}
