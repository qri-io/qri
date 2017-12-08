package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

var (
	validateDsFilepath     string
	validateDsMetaFilepath string
	validateDsName         string
	validateDsUrl          string
	validateDsPassive      bool
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "List errors in a dataset",
	Long: `Validate will load the dataset in question
and check each of it's rows against the constraints listed
in the dataset's fields.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			req := core.NewDatasetRequests(RepoOrClient(false))
			for _, arg := range args {
				rt, ref := dsfs.RefType(arg)
				p := &core.ValidateDatasetParams{}
				switch rt {
				case "path":
					p.Path = datastore.NewKey(ref)
				case "name":
					p.Name = ref
				}

				res := &dataset.Dataset{}
				err := req.Validate(p, res)
				ExitIfErr(err)

				// validation, data, count, err := ds.ValidateData(Cache())
				// ExitIfErr(err)
				// if count > 0 {
				// 	PrintResults(validation, data, dataset.CsvDataFormat)
				// } else {
				// 	PrintSuccess("✔ All good!")
				// }

				PrintSuccess("✔ All good!")
			}

		} else {
			initDataset()
		}
	},
}

func validateDataset() {
	var (
		dataFile, metaFile *os.File
		err                error
	)

	if validateDsFilepath == "" && validateDsUrl == "" {
		ErrExit(fmt.Errorf("please provide either a file or a url argument"))
	} else if validateDsName == "" {
		ErrExit(fmt.Errorf("please provide a --name"))
	}

	dataFile, err = loadFileIfPath(validateDsFilepath)
	ExitIfErr(err)
	metaFile, err = loadFileIfPath(validateDsMetaFilepath)
	ExitIfErr(err)

	req := core.NewDatasetRequests(RepoOrClient(false))

	p := &core.ValidateDatasetParams{
		Name:         validateDsName,
		Url:          validateDsUrl,
		DataFilename: filepath.Base(validateDsFilepath),
	}

	// this is because passing nil to interfaces is bad
	// see: https://golang.org/doc/faq#nil_error
	if dataFile != nil {
		p.Data = dataFile
	}
	if metaFile != nil {
		p.Metadata = metaFile
	}

	ref := &dataset.Dataset{}
	err = req.Validate(p, ref)
	ExitIfErr(err)

	// validation, data, count, err := ds.ValidateData(Cache())
	// ExitIfErr(err)
	// if count > 0 {
	// 	PrintResults(validation, data, dataset.CsvDataFormat)
	// } else {
	// 	PrintSuccess("✔ All good!")
	// }

	PrintSuccess("✔ All good!")
}

func init() {
	validateCmd.Flags().StringVarP(&validateDsName, "name", "n", "", "name to give dataset")
	validateCmd.Flags().StringVarP(&validateDsUrl, "url", "u", "", "url to file to initialize from")
	validateCmd.Flags().StringVarP(&validateDsFilepath, "file", "f", "", "data file to initialize from")
	validateCmd.Flags().StringVarP(&validateDsMetaFilepath, "meta", "m", "", "dataset metadata file")
	validateCmd.Flags().BoolVarP(&validateDsPassive, "passive", "p", false, "disable interactive init")
	RootCmd.AddCommand(validateCmd)
}
