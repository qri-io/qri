package cmd

import (
	"fmt"
	"github.com/qri-io/jsonschema"
	"os"
	"path/filepath"

	// "github.com/ipfs/go-datastore"
	// "github.com/qri-io/dataset"
	// "github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

var (
	validateDsFilepath       string
	validateDsSchemaFilepath string
	validateDsName           string
	validateDsURL            string
	validateDsPassive        bool
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "List errors in a dataset",
	Long: `Validate will load the dataset in question
and check each of it's rows against the constraints listed
in the dataset's fields.`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			dataFile, schemaFile *os.File
			err                  error
		)

		dataFile, err = loadFileIfPath(validateDsFilepath)
		ExitIfErr(err)
		schemaFile, err = loadFileIfPath(validateDsSchemaFilepath)
		ExitIfErr(err)

		req, err := datasetRequests(false)
		ExitIfErr(err)

		p := &core.ValidateDatasetParams{
			Name: validateDsName,
			// URL:          addDsURL,
			DataFilename: filepath.Base(validateDsSchemaFilepath),
		}

		// this is because passing nil to interfaces is bad
		// see: https://golang.org/doc/faq#nil_error
		if dataFile != nil {
			p.Data = dataFile
		}
		if schemaFile != nil {
			p.Schema = schemaFile
		}

		res := []jsonschema.ValError{}
		err = req.Validate(p, &res)
		ExitIfErr(err)
		if len(res) == 0 {
			printSuccess("✔ All good!")
			return
		}

		for i, err := range res {
			fmt.Printf("%d: %s\n", i, err.Error())
		}
	},
}

func init() {
	validateCmd.Flags().StringVarP(&validateDsName, "name", "n", "", "name to give dataset")
	validateCmd.Flags().StringVarP(&validateDsURL, "url", "u", "", "url to file to initialize from")
	validateCmd.Flags().StringVarP(&validateDsFilepath, "file", "f", "", "data file to initialize from")
	validateCmd.Flags().StringVarP(&validateDsSchemaFilepath, "schema", "s", "", "json schema file to use for validation")
	validateCmd.Flags().BoolVarP(&validateDsPassive, "passive", "p", false, "disable interactive init")
	RootCmd.AddCommand(validateCmd)
}
