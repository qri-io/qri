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
			updateMessage = InputText("commit message:", "")
		}

		var datapath string
		if updateFile != "" {
			datapath, err = filepath.Abs(updateFile)
			ExitIfErr(err)
		}
		if updateMetaFile == "" && datapath == "" {
			ErrExit(fmt.Errorf("either a metadata or data option is required"))
		}

		r := GetRepo(false)
		store := GetIpfsFilestore(false)
		req := core.NewDatasetRequests(store, r)

		p := &core.GetDatasetParams{
			Name: args[0],
			Path: datastore.NewKey(args[0]),
		}

		prev := &repo.DatasetRef{}
		err = req.Get(p, prev)
		ExitIfErr(err)

		author, err := r.Profile()
		ExitIfErr(err)

		commit := &core.Commit{
			Author:  author,
			Message: updateMessage,
			Prev:    prev.Path,
		}

		metaFile, err = loadFileIfPath(updateMetaFile)
		ExitIfErr(err)

		if metaFile != nil {
			changes := &dataset.Dataset{}
			err = json.NewDecoder(metaFile).Decode(changes)
			ExitIfErr(err)
			commit.Changes = changes
		}

		res := &repo.DatasetRef{}
		err = req.Update(commit, res)
		ExitIfErr(err)
		PrintSuccess("dataset updated:", res.Path)
	},
}

func init() {
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "data file to updateialize from")
	updateCmd.Flags().StringVarP(&updateMetaFile, "meta", "", "", "dataset metadata updates")
	updateCmd.Flags().StringVarP(&updateMessage, "message", "m", "", "commit message for update")
	updateCmd.Flags().StringVarP(&updateName, "name", "n", "", "name to give dataset")
	RootCmd.AddCommand(updateCmd)
}
