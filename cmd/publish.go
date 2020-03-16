package cmd

import (
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewPublishCommand creates a `qri publish` subcommand for working with configured registries
func NewPublishCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &PublishOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "publish [DATASET]",
		Short: "set dataset publicity",
		Long: `Publish makes your dataset available to others. While online, peers that connect 
to you can only see datasets and versions that you've published. Publishing a 
dataset always makes all previous history entries available, and any updates
to a published dataset will be immediately visible to connected peers.

Publishing a dataset also uploads it to the Qri Cloud registry
(https://qri.cloud/).

Note that publishing makes a single version of the dataset public (by default,
that's the current version). When you update a dataset, those updates need to
be explicitly published to be made public.

Use the --unpublish option to make a dataset private and remove it from a
registry.
`,
		Example: `  # Publish a dataset:
  $ qri publish me/dataset

  # Publish a few datasets:
  $ qri publish me/dataset me/other_dataset

  # Unpublish a dataset:
  $ qri publish --unpublish me/dataset

  # Publish a few dataset on p2p only:
  $ qri publish --no-registry me/dataset_2`,
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

	cmd.Flags().BoolVarP(&o.Unpublish, "unpublish", "", false, "unpublish a dataset")
	cmd.Flags().BoolVarP(&o.NoRegistry, "no-registry", "", false, "don't publish to registry")
	cmd.Flags().BoolVarP(&o.NoPin, "no-pin", "", false, "don't pin dataset to registry")
	cmd.Flags().StringVarP(&o.RemoteName, "remote", "", "", "name of remote to publish to")

	return cmd
}

// PublishOptions encapsulates state for the publish command
type PublishOptions struct {
	ioes.IOStreams

	Refs       *RefSelect
	Unpublish  bool
	NoRegistry bool
	NoPin      bool
	RemoteName string

	DatasetRequests *lib.DatasetRequests
	RemoteMethods   *lib.RemoteMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *PublishOptions) Complete(f Factory, args []string) (err error) {

	if o.DatasetRequests, err = f.DatasetRequests(); err != nil {
		return err
	}

	if o.Refs, err = GetCurrentRefSelect(f, args, 1, nil); err != nil {
		return err
	}

	o.RemoteMethods, err = f.RemoteMethods()
	return
}

// Run executes the publish command
func (o *PublishOptions) Run() error {
	printRefSelect(o.ErrOut, o.Refs)

	p := lib.PublicationParams{
		Ref:        o.Refs.Ref(),
		RemoteName: o.RemoteName,
	}
	var res dsref.Ref
	if o.Unpublish {
		if err := o.RemoteMethods.Unpublish(&p, &res); err != nil {
			return err
		}
		printInfo(o.Out, "unpublished dataset %s", res)
	} else {
		if err := o.RemoteMethods.Publish(&p, &res); err != nil {
			return err
		}
		printInfo(o.Out, "published dataset %s", res)
	}
	return nil
}
