package cmd

import (
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/spf13/cobra"
	"strings"
)

var datasetRemoveCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm"},
	Short:   "remove a dataset from your local namespace based on a resource hash",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			ErrExit(fmt.Errorf("wrong number of arguments for adding a dataset, expected [name]"))
		}
		name := args[0]

		fs := GetIpfsFilestore(false)

		r := GetRepo(false)
		path, err := r.GetPath(name)
		ExitIfErr(err)

		root := datastore.NewKey(strings.TrimSuffix(path.String(), "/"+dsfs.PackageFileDataset.String()))

		err = fs.Delete(root)
		if err != nil {
			PrintWarning(err.Error())
		}
		// ExitIfErr(err)

		err = r.DeleteDataset(path)
		ExitIfErr(err)

		r.DeleteName(name)
		ExitIfErr(err)

		PrintSuccess("removed dataset %s: %s", name, path)
	},
}

func init() {
	RootCmd.AddCommand(datasetRemoveCmd)
}
