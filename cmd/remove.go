package cmd

import (
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/rev"
	"github.com/spf13/cobra"
)

// NewRemoveCommand creates a new `qri remove` cobra command for removing datasets from a local repository
func NewRemoveCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &RemoveOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a dataset from your local repository",
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
  $ qri remove me/annual_pop --all`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.RevisionsText, "revisions", "r", "", "revisions to delete")
	cmd.Flags().BoolVarP(&o.All, "all", "a", false, "synonym for --revisions=all")

	return cmd
}

// RemoveOptions encapsulates state for the remove command
type RemoveOptions struct {
	ioes.IOStreams

	Args []string

	RevisionsText string
	All           bool
	Revision      rev.Rev

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RemoveOptions) Complete(f Factory, args []string) (err error) {
	o.Args = args
	o.DatasetRequests, err = f.DatasetRequests()
	if err != nil {
		return err
	}
	if o.All {
		o.Revision = rev.NewAllRevisions()
	} else {
		if o.RevisionsText == "" {
			return fmt.Errorf("--revisions flag is requried")
		}
		revisions, err := rev.ParseRevs(o.RevisionsText)
		if err != nil {
			return err
		}
		if len(revisions) != 1 {
			return fmt.Errorf("need exactly 1 revision parameter to remove")
		}
		if revisions[0] == nil {
			return fmt.Errorf("invalid nil revision")
		}
		o.Revision = *revisions[0]
	}
	return err
}

// Validate checks that all user input is valid
func (o *RemoveOptions) Validate() error {
	if len(o.Args) == 0 {
		return lib.NewError(lib.ErrBadArgs, "please specify a dataset path or name you would like to remove from your qri node")
	}
	return nil
}

// Run executes the remove command
func (o *RemoveOptions) Run() error {
	for _, arg := range o.Args {
		ref, err := parseCmdLineDatasetRef(arg)
		if err != nil {
			return err
		}

		params := lib.RemoveParams{Ref: &ref, Revision: o.Revision}
		numDeleted := 0
		if err = o.DatasetRequests.Remove(&params, &numDeleted); err != nil {
			if err.Error() == "repo: not found" {
				return lib.NewError(err, fmt.Sprintf("could not find dataset '%s/%s'", ref.Peername, ref.Name))
			}
			return err
		}
		if numDeleted == rev.AllGenerations {
			printSuccess(o.Out, "removed entire dataset '%s'", ref)
		} else {
			printSuccess(o.Out, "removed %d revisions of dataset '%s'", numDeleted, ref)
		}
	}
	return nil
}
