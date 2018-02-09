package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
	"io/ioutil"
)

var (
	saveDataFile       string
	saveMetaFile       string
	saveStructureFile  string
	saveTitle          string
	saveMessage        string
	savePassive        bool
	saveRescursive     bool
	saveShowValidation bool
)

// saveCmd represents the save command
var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "save changes to a dataset",
	Long: `
Save is how you change a dataset, updating one or more of data, metadata, and 
structure. Every time you run save, an entry is added to your dataset’s log 
(which you can see by running “qri log [ref]”). Every time you save, you can 
provide a message about what you changed and why. If you don’t provide a message 
we’ll automatically generate one for you.

Currently you can only save changes to datasets that you control. Tools for 
collaboration are in the works. Sit tight sportsfans.`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			metaFile, dataFile, structureFile *os.File
			err                               error
		)

		if len(args) < 1 {
			ErrExit(fmt.Errorf("please provide the name of an existing dataset so save updates to"))
		}
		if saveMetaFile == "" && saveDataFile == "" && saveStructureFile == "" {
			ErrExit(fmt.Errorf("one of --structure, --meta or --data is required"))
		}

		ref, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		req := core.NewDatasetRequests(getRepo(false), nil)
		save := &core.SaveParams{
			Prev: ref,
			Changes: &dataset.Dataset{
				Commit: &dataset.Commit{
					Title:   saveTitle,
					Message: saveMessage,
				},
			},
		}

		if saveMetaFile != "" {
			metaFile, err = loadFileIfPath(saveMetaFile)
			ExitIfErr(err)
			if metaFile != nil {
				meta := &dataset.Meta{}
				err = json.NewDecoder(metaFile).Decode(meta)
				ExitIfErr(err)
				save.Changes.Meta = meta
			}
		}

		if saveStructureFile != "" {
			structureFile, err = loadFileIfPath(saveStructureFile)
			ExitIfErr(err)
			if structureFile != nil {
				st := &dataset.Structure{}
				err = json.NewDecoder(structureFile).Decode(st)
				ExitIfErr(err)
				save.Changes.Structure = st
			}
		}

		if saveDataFile != "" {
			saveDataFile, err = filepath.Abs(saveDataFile)
			ExitIfErr(err)
			dataFile, err = loadFileIfPath(saveDataFile)
			ExitIfErr(err)
			if dataFile != nil {
				save.DataFilename = filepath.Base(saveDataFile)
				save.Data = dataFile
			}
		} else {
			// TODO - this is silly. dsfs.CreateDataset needs to
			// support being called with a set DataPath and no
			// dataFile
			r := getRepo(false)
			res := repo.DatasetRef{}
			err = req.Get(&ref, &res)
			ExitIfErr(err)

			df, err := dsfs.LoadData(r.Store(), res.Dataset)
			ExitIfErr(err)
			save.Data = df
		}

		res := &repo.DatasetRef{}
		err = req.Save(save, res)
		ExitIfErr(err)
		printSuccess("dataset saved: %s", res)
		if res.Dataset.Structure.ErrCount > 0 {
			printWarning(fmt.Sprintf("this dataset has %d validation errors", res.Dataset.Structure.ErrCount))
			if saveShowValidation {
				printWarning("Validation Error Detail:")
				data, err := ioutil.ReadAll(dataFile)
				ExitIfErr(err)
				errorList := res.Dataset.Structure.Schema.ValidateBytes(data)
				for i, validationErr := range errorList {
					printWarning(fmt.Sprintf("\t%d. %s", i+1, validationErr.Error()))
				}
			}
		}
	},
}

func init() {
	saveCmd.Flags().StringVarP(&saveDataFile, "data", "", "", "data file that forms the dataset")
	saveCmd.Flags().StringVarP(&saveMetaFile, "meta", "", "", "metadata.json file")
	saveCmd.Flags().StringVarP(&saveStructureFile, "structure", "", "", "structure.json file")
	saveCmd.Flags().StringVarP(&saveTitle, "title", "t", "", "title of commit message for save")
	saveCmd.Flags().StringVarP(&saveMessage, "message", "m", "", "commit message for save")
	saveCmd.Flags().BoolVarP(&saveShowValidation, "show-validation", "s", false, "display a list of validation errors upon adding")
	RootCmd.AddCommand(saveCmd)
}
