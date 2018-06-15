package cmd

import (
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewRegistryCommand creates a `qri registry` subcommand for working with configured registries
func NewRegistryCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &RegistryOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "commands for working with a qri registry",
		Long: `Registries are federated public records of datasets and peers.
These records form a public facing central lookup for your datasets, so others
can find them through search tools and via web links. You can use registry 
commands to control how your datasets are published to registries, opting out
on a dataset-by-dataset basis.

By default qri is configured to publish to https://registry.qri.io,
the main public collection of datasets & peers. "qri add" and "qri update"
default to publishing to a registry as part of dataset creation unless run 
with the "no-registry" flag.

Unpublished dataset info will be held locally so you can still interact
with it. And your datasets will be available to others peers when you run 
"qri connect", but will not show up in search results, and will not be 
displayed on lists of registry datasets.

Qri is designed to work without a registry should you want to opt out of
centralized listing entirely, but know that peers who *do* participate in
registries may choose to deprioritize connections with you. Opting out of a
registry entirely is better left to advanced users.

You can opt out of registries entirely by running:
$ qri config set registry.location ""`,

		Annotations: map[string]string{
			"group": "network",
		},
	}

	// publishCmd represents the publish command
	publish := &cobra.Command{
		Use:   "publish",
		Short: "publish dataset info to the registry",
		Example: `  Publish a dataset you've created to the registry:
  $ qri registry publish me/dataset_name`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Publish())
		},
	}

	// unpublishCmd represents the unpublish command
	unpublish := &cobra.Command{
		Use:   "unpublish",
		Short: "remove dataset info from the registry",
		Example: `  Remove a dataset from the registry:
  $ qri registry unpublish me/dataset_name`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Unpublish())
		},
	}

	cmd.AddCommand(publish, unpublish)
	return cmd
}

// RegistryOptions encapsulates state for the registry command & subcommands
type RegistryOptions struct {
	IOStreams

	Refs []string

	RegistryRequests *core.RegistryRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RegistryOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.RegistryRequests, err = f.RegistryRequests()
	return
}

// Publish executes the publish command
func (o *RegistryOptions) Publish() error {
	var res bool
	for _, arg := range o.Refs {
		ref, err := repo.ParseDatasetRef(arg)
		if err != nil {
			return err
		}

		if err = o.RegistryRequests.Publish(&ref, &res); err != nil {
			return err
		}
		printInfo(o.Out, "published dataset %s", ref)
	}
	return nil
}

// Unpublish executes the unpublish command
func (o *RegistryOptions) Unpublish() error {
	var res bool
	for _, arg := range o.Refs {
		ref, err := repo.ParseDatasetRef(arg)
		if err != nil {
			return err
		}

		if err = o.RegistryRequests.Unpublish(&ref, &res); err != nil {
			return err
		}
		printInfo(o.Out, "unpublished dataset %s", ref)
	}
	return nil
}
