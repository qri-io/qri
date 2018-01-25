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
	saveFile       string
	saveMetaFile   string
	saveTitle      string
	saveMessage    string
	saveName       string
	savePassive    bool
	saveRescursive bool
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
			metaFile, dataFile *os.File
			err                error
		)
		if saveMessage == "" {
			saveMessage = inputText("commit message:", "")
		}

		var datapath string
		if saveFile != "" {
			datapath, err = filepath.Abs(saveFile)
			ExitIfErr(err)
		}
		if saveMetaFile == "" && datapath == "" {
			ErrExit(fmt.Errorf("either a metadata or data option is required"))
		}

		req, err := datasetRequests(false)
		ExitIfErr(err)

		p := &core.GetDatasetParams{
			Name: saveName,
			// Path: datastore.NewKey(saveName),
		}

		prev := &repo.DatasetRef{}
		err = req.Get(p, prev)
		ExitIfErr(err)

		// author, err := r.Profile()
		// ExitIfErr(err)

		save := &core.UpdateParams{}

		metaFile, err = loadFileIfPath(saveMetaFile)
		ExitIfErr(err)

		save.Changes = prev.Dataset
		save.Changes.PreviousPath = prev.Path.String()

		if metaFile != nil {
			changes := &dataset.Dataset{}
			err = json.NewDecoder(metaFile).Decode(changes)
			ExitIfErr(err)
			save.Changes = changes
		}

		dataFile, err = loadFileIfPath(saveFile)
		ExitIfErr(err)

		if dataFile != nil {
			save.DataFilename = filepath.Base(saveFile)
			save.Data = dataFile
		}

		save.Changes.Commit.Assign(&dataset.Commit{
			// Author:  &dataset.User{ID: author.ID, Email: author.Email},
			Title:   saveTitle,
			Message: saveMessage,
		})
		save.Changes.PreviousPath = prev.Path.String()

		res := &repo.DatasetRef{}
		err = req.Update(save, res)
		ExitIfErr(err)
		printSuccess("dataset saved: %s", res.Path)
	},
}

func init() {
	saveCmd.Flags().StringVarP(&saveFile, "file", "f", "", "data file to saveialize from")
	saveCmd.Flags().StringVarP(&saveMetaFile, "meta", "", "", "dataset metadata saves")
	saveCmd.Flags().StringVarP(&saveTitle, "title", "t", "", "title of commit message for save")
	saveCmd.Flags().StringVarP(&saveMessage, "message", "m", "", "commit message for save")
	saveCmd.Flags().StringVarP(&saveName, "name", "n", "", "name to give dataset")
	RootCmd.AddCommand(saveCmd)
}
