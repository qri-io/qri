package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
	"io/ioutil"
)

var (
	addDsFilepath          string
	addDsMetaFilepath      string
	addDsStructureFilepath string
	addDsName              string
	addDsURL               string
	addDsPassive           bool
	addDsShowValidation    bool
	addDsPrivate           bool
)

var datasetAddCmd = &cobra.Command{
	Use:        "add",
	Short:      "Add a dataset",
	SuggestFor: []string{"init"},
	Annotations: map[string]string{
		"group": "dataset",
	},
	Long: `
Add creates a new dataset from data you supply. Please note that all data added 
to qri is made public on the distributed web when you run qri connect.

When adding data, you can supply metadata and dataset structure, but it’s not 
required. qri does what it can to infer the details you don’t provide. 
add currently supports two data formats:
- CSV  (Comma Separated Values)
- JSON (Javascript Object Notation)
- CBOR (Concise Binary Object Representation)

Once you’ve added data, you can use the export command to pull the data out of 
qri, change the data outside of qri, and use the save command to record those 
changes to qri.`,
	Example: `  add a new dataset named annual_pop:
  $ qri add --data data.csv me/annual_pop

  create a dataset with a metadata and data file:
  $ qri add --meta meta.json --data comics.csv me/comic_characters`,
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {

		ingest := (addDsFilepath != "" || addDsMetaFilepath != "" || addDsStructureFilepath != "" || addDsURL != "")

		if len(args) == 0 {
			ErrExit(fmt.Errorf("specify the location of a dataset to add"))
		} else if ingest && len(args) != 1 {
			ErrExit(fmt.Errorf("adding datasets with --structure, --meta, or --data requires exactly 1 argument for the new dataset name"))
		}

		if ingest {
			ref, err := repo.ParseDatasetRef(args[0])
			ExitIfErr(err)

			initDataset(ref)
			return
		}

		for _, arg := range args {
			if addDsPrivate {
				ErrExit(fmt.Errorf("option to make dataset private not yet implimented, refer to https://github.com/qri-io/qri/issues/291 for updates"))
			}

			ref, err := repo.ParseDatasetRef(arg)
			ExitIfErr(err)

			req, err := datasetRequests(true)
			ExitIfErr(err)

			res := repo.DatasetRef{}
			err = req.Add(&ref, &res)
			ExitIfErr(err)
			printInfo("Successfully added dataset %s", ref)
		}
	},
}

func initDataset(name repo.DatasetRef) {
	var (
		dataFile, metaFile, structureFile *os.File
		err                               error
	)

	if addDsFilepath == "" && addDsURL == "" || addDsFilepath != "" && addDsURL != "" {
		ErrExit(fmt.Errorf("please provide either a file or a url argument"))
	}

	dataFile, err = loadFileIfPath(addDsFilepath)
	ExitIfErr(err)
	metaFile, err = loadFileIfPath(addDsMetaFilepath)
	ExitIfErr(err)
	structureFile, err = loadFileIfPath(addDsStructureFilepath)
	ExitIfErr(err)

	p := &core.InitParams{
		Peername:     name.Peername,
		Name:         name.Name,
		URL:          addDsURL,
		DataFilename: filepath.Base(addDsFilepath),
		Private:      addDsPrivate,
	}

	// this is because passing nil to interfaces is bad
	// see: https://golang.org/doc/faq#nil_error
	if dataFile != nil {
		p.Data = dataFile
	}
	if metaFile != nil {
		p.Metadata = metaFile
	}
	if structureFile != nil {
		p.Structure = structureFile
	}

	req, err := datasetRequests(false)
	ExitIfErr(err)

	ref := repo.DatasetRef{}
	err = req.Init(p, &ref)
	ExitIfErr(err)

	if ref.Dataset.Structure.ErrCount > 0 {
		printWarning(fmt.Sprintf("this dataset has %d validation errors", ref.Dataset.Structure.ErrCount))
		if addDsShowValidation {
			printWarning("Validation Error Detail:")
			data, err := ioutil.ReadAll(dataFile)
			ExitIfErr(err)
			ds, err := ref.DecodeDataset()
			ErrExit(err)
			errorList, err := ds.Structure.Schema.ValidateBytes(data)
			ExitIfErr(err)
			for i, validationErr := range errorList {
				printWarning(fmt.Sprintf("\t%d. %s", i+1, validationErr.Error()))
			}
		}
	}

	ref.Peername = "me"
	printSuccess("added new dataset %s", ref)
}

func init() {
	datasetAddCmd.Flags().StringVarP(&addDsURL, "url", "", "", "url of file to initialize from")
	datasetAddCmd.Flags().StringVarP(&addDsFilepath, "data", "", "", "data file to initialize from")
	datasetAddCmd.Flags().StringVarP(&addDsStructureFilepath, "structure", "", "", "dataset structure JSON file")
	datasetAddCmd.Flags().StringVarP(&addDsMetaFilepath, "meta", "", "", "dataset metadata JSON file")
	datasetAddCmd.Flags().BoolVarP(&addDsPrivate, "private", "", false, "make dataset private. WARNING: not yet implimented. Please refer to https://github.com/qri-io/qri/issues/291 for updates")
	datasetAddCmd.Flags().BoolVarP(&addDsShowValidation, "show-validation", "s", false, "display a list of validation errors upon adding")
	RootCmd.AddCommand(datasetAddCmd)
}
