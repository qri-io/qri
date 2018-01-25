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
	Short:   "remove a dataset from your local repository",
	Long: `
remove gets rid of datasets. After running remove, qri will no longer list your 
dataset as being available locally. By default remove frees up the space taken 
up by the dataset, but not right away. This is because the IPFS repo that’s 
storing the data will need to garbage-collect that data when it’s good & ready, 
which could be anytime. If you’re running low on space, garbage collection will 
be sooner. 

Keep in mind that by default your IPFS repo is capped at 10GB in size, if you
adjust this cap using IPFS, qri will respect it.

In the future we’ll add a flag that’ll force immediate removal of a dataset from
both qri & IPFS. Promise.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			ErrExit(fmt.Errorf("please specify a dataset path or name to get the info of"))
		}

		req, err := datasetRequests(false)
		ExitIfErr(err)

		for _, arg := range args {
			rt, ref := dsfs.RefType(arg)
			p := &core.RemoveParams{}
			switch rt {
			case "path":
				p.Path = datastore.NewKey(ref)
			case "name":
				p.Name = ref
			}
			res := false
			err := req.Remove(p, &res)
			ExitIfErr(err)
			printSuccess("removed dataset %s", ref)
		}
	},
}

func init() {
	RootCmd.AddCommand(datasetRemoveCmd)
}
