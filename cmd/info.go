package cmd

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"get", "describe"},
	Short:   "Show info about a dataset",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			ErrExit(fmt.Errorf("please specify a dataset path or name to get the info of"))
		}

		req := core.NewDatasetRequests(GetRepo(false))

		for i, arg := range args {
			rt, ref := dsfs.RefType(arg)
			p := &core.GetDatasetParams{}
			switch rt {
			case "path":
				p.Path = datastore.NewKey(ref)
			case "name":
				p.Name = ref
			}
			res := &repo.DatasetRef{}
			err := req.Get(p, res)
			ExitIfErr(err)
			PrintDatasetRefInfo(i, res)
		}
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
}
