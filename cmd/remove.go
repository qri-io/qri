package cmd

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

var datasetRemoveCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm"},
	Short:   "remove a dataset from your local namespace based on a resource hash",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			ErrExit(fmt.Errorf("please specify a dataset path or name to get the info of"))
		}

		req := core.NewDatasetRequests(GetRepo(false))

		for _, arg := range args {
			rt, ref := dsfs.RefType(arg)
			p := &core.DeleteParams{}
			switch rt {
			case "path":
				p.Path = datastore.NewKey(ref)
			case "name":
				p.Name = ref
			}
			res := false
			err := req.Delete(p, &res)
			ExitIfErr(err)
			PrintSuccess("removed dataset %s", ref)
		}
	},
}

func init() {
	RootCmd.AddCommand(datasetRemoveCmd)
}
