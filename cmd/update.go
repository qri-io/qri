package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/dataset"
	"os"
	"path/filepath"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	updateFile       string
	updateMetaFile   string
	updateTitle      string
	updateMessage    string
	updateName       string
	updatePassive    bool
	updateRescursive bool
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a dataset, changing metadata and/or data",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			metaFile *os.File
			err      error
		)
		if updateMessage == "" {
			updateMessage = inputText("commit message:", "")
		}

		var datapath string
		if updateFile != "" {
			datapath, err = filepath.Abs(updateFile)
			ExitIfErr(err)
		}
		if updateMetaFile == "" && datapath == "" {
			ErrExit(fmt.Errorf("either a metadata or data option is required"))
		}

		req, err := datasetRequests(false)
		ExitIfErr(err)

		p := &core.GetDatasetParams{
			Name: args[0],
			Path: datastore.NewKey(args[0]),
		}

		prev := &repo.DatasetRef{}
		err = req.Get(p, prev)
		ExitIfErr(err)

		author, err := r.Profile()
		ExitIfErr(err)

		update := &core.UpdateParams{}

		metaFile, err = loadFileIfPath(updateMetaFile)
		ExitIfErr(err)

		if metaFile != nil {
			changes := &dataset.Dataset{}
			err = json.NewDecoder(metaFile).Decode(changes)
			ExitIfErr(err)
			update.Changes = changes
		}

		update.Changes.Commit.Assign(&dataset.CommitMsg{
			Author:  &dataset.User{ID: author.ID, Email: author.Email},
			Title:   updateTitle,
			Message: updateMessage,
		})
		update.Changes.Previous = prev.Path

		res := &repo.DatasetRef{}
		err = req.Update(update, res)
		ExitIfErr(err)
		printSuccess("dataset updated:", res.Path)
	},
}

func init() {
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "data file to updateialize from")
	updateCmd.Flags().StringVarP(&updateMetaFile, "meta", "", "", "dataset metadata updates")
	updateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "title of commit message for update")
	updateCmd.Flags().StringVarP(&updateMessage, "message", "m", "", "commit message for update")
	updateCmd.Flags().StringVarP(&updateName, "name", "n", "", "name to give dataset")
	RootCmd.AddCommand(updateCmd)
}
