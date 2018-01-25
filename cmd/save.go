package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/dataset"
	"os"
	"path/filepath"

	// "github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	saveDataFile      string
	saveMetaFile      string
	saveStructureFile string
	saveTitle         string
	saveMessage       string
	savePassive       bool
	saveRescursive    bool
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
			ErrExit(fmt.Errorf("either a metadata or data option is required"))
		}

		ref, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		req, err := datasetRequests(false)
		ExitIfErr(err)

		// TODO - need to make sure users aren't forking by referncing commits other than tip
		p := &core.GetDatasetParams{
			Name: ref.Name,
			Path: ref.Path,
		}

		prev := &repo.DatasetRef{}
		err = req.Get(p, prev)
		ExitIfErr(err)

		save := &core.SaveParams{}
		save.Changes = prev.Dataset
		save.Changes.PreviousPath = prev.Path.String()

		if saveMetaFile != "" {
			fmt.Println(saveMetaFile)
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
		}

		save.Changes.Commit.Assign(&dataset.Commit{
			Title:   saveTitle,
			Message: saveMessage,
		})
		save.Changes.PreviousPath = prev.Path.String()

		res := &repo.DatasetRef{}
		err = req.Save(save, res)
		ExitIfErr(err)
		printSuccess("dataset saved: %s", res.Path)
	},
}

func init() {
	saveCmd.Flags().StringVarP(&saveDataFile, "data", "", "", "data file that forms the dataset")
	saveCmd.Flags().StringVarP(&saveMetaFile, "meta", "", "", "metadata.json file")
	saveCmd.Flags().StringVarP(&saveStructureFile, "structure", "", "", "structure.json file")

	saveCmd.Flags().StringVarP(&saveTitle, "title", "t", "", "title of commit message for save")
	saveCmd.Flags().StringVarP(&saveMessage, "message", "m", "", "commit message for save")

	// saveCmd.Flags().BoolVarP(&exportCmdDataset, "dataset", "", false, "export full dataset package")
	// saveCmd.Flags().BoolVarP(&exportCmdMeta, "meta", "m", false, "export dataset metadata file")
	// saveCmd.Flags().BoolVarP(&exportCmdStructure, "structure", "s", false, "export dataset structure file")
	// saveCmd.Flags().BoolVarP(&exportCmdData, "data", "d", true, "export dataset data file")
	// saveCmd.Flags().BoolVarP(&exportCmdTransform, "transform", "t", false, "export dataset transform file")
	// saveCmd.Flags().BoolVarP(&exportCmdVis, "vis-conf", "c", false, "export viz config file")

	RootCmd.AddCommand(saveCmd)
}
