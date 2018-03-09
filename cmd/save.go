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
	saveDataFile       string
	saveURL            string
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
structure. You can also update your data via url. Every time you run save, 
an entry is added to your dataset’s log 
(which you can see by running “qri log [ref]”). Every time you save, you can 
provide a message about what you changed and why. If you don’t provide a message 
we’ll automatically generate one for you.

Currently you can only save changes to datasets that you control. Tools for 
collaboration are in the works. Sit tight sportsfans.`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			dataFile, metaFile, structureFile *os.File
			err                               error
		)

		if len(args) < 1 {
			ErrExit(fmt.Errorf("please provide the name of an existing dataset so save updates to"))
		}
		if saveMetaFile == "" && saveDataFile == "" && saveStructureFile == "" {
			ErrExit(fmt.Errorf("one of --structure, --meta or --data or --url is required"))
		}

		ref, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		dataFile, err = loadFileIfPath(saveDataFile)
		ExitIfErr(err)
		metaFile, err = loadFileIfPath(saveMetaFile)
		ExitIfErr(err)
		structureFile, err = loadFileIfPath(saveStructureFile)
		ExitIfErr(err)

		save := &core.SaveParams{
			Name:              ref.Name,
			Peername:          ref.Peername,
			URL:               saveURL,
			Title:             saveTitle,
			Message:           saveMessage,
			DataFilename:      filepath.Base(saveDataFile),
			MetadataFilename:  filepath.Base(saveMetaFile),
			StructureFilename: filepath.Base(saveStructureFile),
		}

		if dataFile != nil {
			save.Data = dataFile
		}
		if metaFile != nil {
			save.Metadata = metaFile
		}
		if structureFile != nil {
			save.Structure = structureFile
		}

		req, err := datasetRequests(false)
		ExitIfErr(err)

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
				errorList, err := res.Dataset.Structure.Schema.ValidateBytes(data)
				ExitIfErr(err)
				for i, validationErr := range errorList {
					printWarning(fmt.Sprintf("\t%d. %s", i+1, validationErr.Error()))
				}
			}
		}
	},
}

func init() {
	saveCmd.Flags().StringVarP(&saveDataFile, "data", "", "", "data file that forms the dataset")
	saveCmd.Flags().StringVarP(&saveURL, "url", "", "", "url that data file can be updated from")
	saveCmd.Flags().StringVarP(&saveMetaFile, "meta", "", "", "metadata.json file")
	saveCmd.Flags().StringVarP(&saveStructureFile, "structure", "", "", "structure.json file")
	saveCmd.Flags().StringVarP(&saveTitle, "title", "t", "", "title of commit message for save")
	saveCmd.Flags().StringVarP(&saveMessage, "message", "m", "", "commit message for save")
	saveCmd.Flags().BoolVarP(&saveShowValidation, "show-validation", "s", false, "display a list of validation errors upon adding")
	RootCmd.AddCommand(saveCmd)
}
