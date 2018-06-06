package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	addDsFile           string
	addDsDataPath       string
	addDsName           string
	addDsTitle          string
	addDsMessage        string
	addDsPassive        bool
	addDsShowValidation bool
	addDsPrivate        bool
	addDsSecrets        []string
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
add currently supports three data formats:
- CSV  (Comma Separated Values)
- JSON (Javascript Object Notation)
- CBOR (Concise Binary Object Representation)

Once you’ve added data, you can use the export command to pull the data out of 
qri, change the data outside of qri, and use the save command to record those 
changes to qri.`,
	Example: `  add a new dataset named annual_pop:
  $ qri add --data data.csv me/annual_pop

  create a dataset with a dataset data file:
  $ qri add --file dataset.yaml --data comics.csv me/comic_characters`,
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {

		ingest := (addDsFile != "" || addDsDataPath != "")

		if ingest {
			var arg string
			if len(args) == 1 {
				arg = args[0]
			}
			ref, _ := repo.ParseDatasetRef(arg)
			initDataset(ref, cmd)
			return
		}

		for _, arg := range args {
			ref, err := repo.ParseDatasetRef(arg)
			ExitIfErr(err)

			req, err := datasetRequests(true)
			ExitIfErr(err)

			res := repo.DatasetRef{}
			err = req.Add(&ref, &res)
			ExitIfErr(err)
			printDatasetRefInfo(1, res)
			printInfo("Successfully added dataset %s", ref)
		}
	},
}

func initDataset(name repo.DatasetRef, cmd *cobra.Command) {
	var err error

	dsp := &dataset.DatasetPod{}
	if addDsFile != "" {
		f, err := os.Open(addDsFile)
		ExitIfErr(err)

		switch strings.ToLower(filepath.Ext(addDsFile)) {
		case ".yaml", ".yml":
			data, err := ioutil.ReadAll(f)
			ExitIfErr(err)
			err = dsutil.UnmarshalYAMLDatasetPod(data, dsp)
			ExitIfErr(err)
		case ".json":
			err = json.NewDecoder(f).Decode(dsp)
			ExitIfErr(err)
		}
	}

	if name.Peername != "" {
		dsp.Name = name.Name
	}
	if name.Peername != "" {
		dsp.Peername = name.Peername
	}
	if addDsDataPath != "" {
		addDsDataPath, err = filepath.Abs(addDsDataPath)
		ExitIfErr(err)
		dsp.DataPath = addDsDataPath
	}
	if dsp.Transform != nil {
		if addDsSecrets != nil {
			if !confirm(`
Warning: You are providing secrets to a dataset transformation.
Never provide secrets to a transformation you do not trust.
continue?`, true) {
				return
			}
			dsp.Transform.Secrets, err = parseSecrets(addDsSecrets...)
			ExitIfErr(err)
		}
		if dsp.Transform.ScriptPath != "" {
			dsp.Transform.ScriptPath, err = filepath.Abs(dsp.Transform.ScriptPath)
			ExitIfErr(err)
		}
	}

	if dsp.Commit == nil && (addDsTitle != "" || addDsMessage != "") {
		dsp.Commit = &dataset.CommitPod{}
	}

	if addDsTitle != "" {
		dsp.Commit.Title = addDsTitle
	}

	if addDsMessage != "" {
		dsp.Commit.Message = addDsMessage
	}

	p := &core.SaveParams{
		Dataset: dsp,
		Private: addDsPrivate,
	}

	req, err := datasetRequests(false)
	ExitIfErr(err)

	ref := repo.DatasetRef{}
	err = req.Init(p, &ref)
	ExitIfErr(err)

	if ref.Dataset.Structure.ErrCount > 0 {
		printWarning(fmt.Sprintf("this dataset has %d validation errors", ref.Dataset.Structure.ErrCount))

		// TODO - restore.
		// if addDsShowValidation {
		// 	printWarning("Validation Error Detail:")
		// 	data, err := ioutil.ReadAll(dataFile)
		// 	ExitIfErr(err)
		// 	ds, err := ref.DecodeDataset()
		// 	ErrExit(err)
		// 	errorList, err := ds.Structure.Schema.ValidateBytes(data)
		// 	ExitIfErr(err)
		// 	for i, validationErr := range errorList {
		// 		printWarning(fmt.Sprintf("\t%d. %s", i+1, validationErr.Error()))
		// 	}
		// }
	}

	ref.Peername = "me"
	printSuccess("added new dataset %s", ref)
}

func init() {
	datasetAddCmd.Flags().StringVarP(&addDsFile, "file", "f", "", "dataset data file in either yaml or json format")
	datasetAddCmd.Flags().StringVarP(&addDsDataPath, "data", "d", "", "path to file or url to initialize from")
	datasetAddCmd.Flags().StringVarP(&addDsTitle, "title", "t", "", "commit title")
	datasetAddCmd.Flags().StringVarP(&addDsMessage, "message", "m", "", "commit message")
	datasetAddCmd.Flags().BoolVarP(&addDsPrivate, "private", "", false, "make dataset private. WARNING: not yet implimented. Please refer to https://github.com/qri-io/qri/issues/291 for updates")
	datasetAddCmd.Flags().StringSliceVar(&addDsSecrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	// datasetAddCmd.Flags().BoolVarP(&addDsShowValidation, "show-validation", "s", false, "display a list of validation errors upon adding")
	RootCmd.AddCommand(datasetAddCmd)
}
