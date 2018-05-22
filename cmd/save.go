package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
	"io/ioutil"
)

var (
	saveDataPath       string
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
	Use:     "save",
	Aliases: []string{"update"},
	Short:   "save changes to a dataset",
	Long: `
Save is how you change a dataset, updating one or more of data, metadata, and 
structure. You can also update your data via url. Every time you run save, 
an entry is added to your dataset’s log 
(which you can see by running “qri log [ref]”). Every time you save, you can 
provide a message about what you changed and why. If you don’t provide a message 
qri will automatically generate one for you.

Currently you can only save changes to datasets that you control. Tools for 
collaboration are in the works. Sit tight sportsfans.`,
	Annotations: map[string]string{
		"group": "dataset",
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			ErrExit(fmt.Errorf("please provide the name of an existing dataset to save updates to"))
		}
		// if saveMetaFile == "" && saveDataPath == "" && saveStructureFile == "" {
		// 	ErrExit(fmt.Errorf("one of --structure, --meta or --data or --url is required"))
		// }

		ref, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		// metaFile, err = loadFileIfPath(saveMetaFile)
		// ExitIfErr(err)
		// structureFile, err = loadFileIfPath(saveStructureFile)
		// ExitIfErr(err)

		p := &core.SaveParams{
			Name:     ref.Name,
			Title:    saveTitle,
			Message:  saveMessage,
			Peername: ref.Peername,
			DataURL:  saveURL,
			DataPath: saveDataPath,
			// MetadataFilename:  filepath.Base(saveMetaFile),
			// StructureFilename: filepath.Base(saveStructureFile),
		}

		if dspath, err := cmd.Flags().GetString("dataset"); err == nil && dspath != "" {
			ds := &dataset.DatasetPod{}
			f, err := os.Open(dspath)
			ExitIfErr(err)

			switch strings.ToLower(filepath.Ext(dspath)) {
			case ".yaml", ".yml":
				data, err := ioutil.ReadAll(f)
				ExitIfErr(err)
				// err = UnmarshalYAML(data, ds)
				err = dsutil.UnmarshalYAMLDatasetPod(data, ds)
				ExitIfErr(err)
			case ".json":
				err = json.NewDecoder(f).Decode(ds)
				ExitIfErr(err)
			}
			p.Dataset = ds
		}

		req, err := datasetRequests(false)
		ExitIfErr(err)

		res := &repo.DatasetRef{}
		err = req.Save(p, res)
		ExitIfErr(err)

		printSuccess("dataset saved: %s", res)
		if res.Dataset.Structure.ErrCount > 0 {
			printWarning(fmt.Sprintf("this dataset has %d validation errors", res.Dataset.Structure.ErrCount))

			// TODO - restore. This should read from the created dataset instead of input data
			// if saveShowValidation {
			// 	printWarning("Validation Error Detail:")
			// 	data, err := ioutil.ReadAll(dataFile)
			// 	ExitIfErr(err)
			// 	ds, err := res.DecodeDataset()
			// 	ExitIfErr(err)
			// 	errorList, err := ds.Structure.Schema.ValidateBytes(data)
			// 	ExitIfErr(err)
			// 	for i, validationErr := range errorList {
			// 		printWarning(fmt.Sprintf("\t%d. %s", i+1, validationErr.Error()))
			// 	}
			// }
		}
	},
}

func init() {
	saveCmd.Flags().StringP("dataset", "", "", "dataset data file (yaml or json)")
	saveCmd.Flags().StringVarP(&saveTitle, "title", "t", "", "title of commit message for save")
	saveCmd.Flags().StringVarP(&saveMessage, "message", "m", "", "commit message for save")
	saveCmd.Flags().StringVarP(&saveURL, "url", "", "", "url that data file can be updated from")
	saveCmd.Flags().StringVarP(&saveDataPath, "data", "", "", "data file that forms the dataset")
	// saveCmd.Flags().StringVarP(&saveMetaFile, "meta", "", "", "metadata.json file")
	// saveCmd.Flags().StringVarP(&saveStructureFile, "structure", "", "", "structure.json file")
	saveCmd.Flags().BoolVarP(&saveShowValidation, "show-validation", "s", false, "display a list of validation errors upon adding")
	RootCmd.AddCommand(saveCmd)
}
