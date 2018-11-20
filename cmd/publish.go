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
to a published dataset will be immideately visible to connected peers.
`,
		Example: `  # publish a dataset
  $ qri publish me/dataset

  # publish a few datasets
  $ qri publish me/dataset me/other_dataset

  # unpublish a dataset
  $ qri publish -unpublish me/dataset`,
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

	return cmd
}

// PublishOptions encapsulates state for the publish command
type PublishOptions struct {
	ioes.IOStreams

	Refs       []string
	Unpublish  bool
	NoRegistry bool

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *PublishOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the publish command
func (o *PublishOptions) Run() error {
	var res bool

	for _, arg := range o.Refs {
		ref, err := repo.ParseDatasetRef(arg)
		if err != nil {
			return err
		}

		ref.Published = !o.Unpublish
		p := &lib.SetPublishStatusParams{
			Ref:            &ref,
			UpdateRegistry: !o.NoRegistry,
		}

		if err = o.DatasetRequests.SetPublishStatus(p, &res); err != nil {
			return err
		}
		printInfo(o.Out, "published dataset %s", ref)
	}
	return nil
}
