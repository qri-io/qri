package cmd

import (
	"context"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewPushCommand creates a `qri push` subcommand
func NewPushCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &PushOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "push DATASET [DATASET...] [flags]",
		Short:   "send a dataset to a remote",
		Aliases: []string{"publish"},
		Long: `Push sends datasets to a remote qri node. A push updates the dataset log on the
remote and sends one version of dataset data to the remote. To push multiple
dataset versions, run push multiple times, specifying the version hash to push.

If no remote is specified, qri pushes to the registry.`,
		Example: `  # push a dataset to the registry
  $ qri push me/dataset

  # push a specific version of a dataset to the registry:
  $ qri push me/dataset@/ipfs/QmHashOfVersion`,
		Annotations: map[string]string{
			"group": "network",
		},
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().BoolVarP(&o.Logs, "logs", "", false, "send only dataset history")
	cmd.Flags().StringVarP(&o.RemoteName, "remote", "", "", "name of remote to push to")

	return cmd
}

// PushOptions encapsulates state for the push command
type PushOptions struct {
	ioes.IOStreams

	Refs       *RefSelect
	Logs       bool
	RemoteName string

	inst *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *PushOptions) Complete(f Factory, args []string) (err error) {
	if o.inst, err = f.Instance(); err != nil {
		return err
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, 1, nil); err != nil {
		return err
	}
	return nil
}

// Run executes the push command
func (o *PushOptions) Run() error {
	ctx := context.TODO()
	for _, ref := range o.Refs.RefList() {
		p := lib.PushParams{
			Ref:    ref,
			Remote: o.RemoteName,
		}

		res, err := o.inst.Remote().Push(ctx, &p)
		if err != nil {
			return err
		}
		printInfo(o.Out, "pushed dataset %s", res)
	}

	return nil
}
