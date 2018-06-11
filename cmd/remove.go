package cmd

import (
	"fmt"

	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

func NewRemoveCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &RemoveOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm", "delete"},
		Short:   "remove a dataset from your local repository",
		Long: `
Remove gets rid of a dataset from your qri node. After running remove, qri will 
no longer list your dataset as being available locally. By default, remove frees
up the space taken up by the dataset, but not right away. The IPFS repo that’s 
storing the data will need to garbage-collect that data when it’s good & ready, 
which could be anytime. If you’re running low on space, garbage collection will 
be sooner. 

Keep in mind that by default your IPFS repo is capped at 10GB in size, if you
adjust this cap using IPFS, qri will respect it.

In the future we’ll add a flag that’ll force immediate removal of a dataset from
both qri & IPFS. Promise.`,
		Example: `  remove a dataset named annual_pop:
  $ qri remove me/annual_pop`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Run())
		},
	}

	return cmd
}

type RemoveOptions struct {
	IOStreams

	Args []string

	DatasetRequests *core.DatasetRequests
}

func (o *RemoveOptions) Complete(f Factory, args []string) (err error) {
	o.Args = args
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

func (o *RemoveOptions) Run() error {
	if len(o.Args) == 0 {
		return fmt.Errorf("please specify a dataset path or name to get the info of")
	}

	for _, arg := range o.Args {
		ref, err := repo.ParseDatasetRef(arg)
		if err != nil {
			return err
		}

		res := false
		if err = o.DatasetRequests.Remove(&ref, &res); err != nil {
			return err
		}
		printSuccess(o.Out, "removed dataset %s", ref)
	}
	return nil
}
