package cmd

import (
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewPublishCommand creates a `qri publish` subcommand for working with configured registries
func NewPublishCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &PublishOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "set dataset publicity",
		Long: `Publish makes your dataset available to others. While online, peers that connect 
to you can only see datasets and versions that you've published. Publishing a 
dataset always makes all previous history entries available, and any updates
to a published dataset will be immediately visible to connected peers.
`,
		Example: `  # publish a dataset
  $ qri publish me/dataset

  # publish a few datasets
  $ qri publish me/dataset me/other_dataset

  # unpublish a dataset
  $ qri publish -unpublish me/dataset

  # publish a few dataset on p2p only
  $ qri publish --no-registry me/dataset_2`,
		Annotations: map[string]string{
			"group": "network",
		},
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

	Refs       []string
	Unpublish  bool
	NoRegistry bool
	NoPin      bool
	RemoteName string

	DatasetRequests *lib.DatasetRequests
	RemoteRequests  *lib.RemoteRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *PublishOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.DatasetRequests, err = f.DatasetRequests()
	o.RemoteRequests, err = f.RemoteRequests()
	return
}

// Run executes the publish command
func (o *PublishOptions) Run() error {
	for _, ref := range o.Refs {
		if o.RemoteName != "" {
			// Publish for a "Remote".
			p := lib.PushParams{
				Ref:        ref,
				RemoteName: o.RemoteName,
			}
			var res bool
			if err := o.RemoteRequests.PushToRemote(&p, &res); err != nil {
				return err
			}
			// TODO(dlong): Check if the operation succeeded or failed. Perform dsync.
			return nil
		}

		// Publish for the legacy Registry server.
		p := lib.SetPublishStatusParams{
			Ref:               ref,
			PublishStatus:     !o.Unpublish,
			UpdateRegistry:    !o.NoRegistry,
			UpdateRegistryPin: !o.NoPin,
		}

		var publishedRef repo.DatasetRef
		if err := o.DatasetRequests.SetPublishStatus(&p, &publishedRef); err != nil {
			return err
		}
		printInfo(o.Out, "published dataset %s", publishedRef)
	}
	return nil
}
