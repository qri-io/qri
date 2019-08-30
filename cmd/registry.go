package cmd

import (
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewRegistryCommand creates a `qri registry` subcommand for working with configured registries
// TODO - registry command is currently removed in favor of the newer "qri publish" command
// we should consider refactoring this code (espcially it's documentation) &
// use it for registry-specific publication & search interaction
func NewRegistryCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &RegistryOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Commands for working with a qri registry",
		Long: `
Registries are federated public records of datasets and peers.
These records form a public facing central lookup for your datasets, so others
can find them through search tools and via web links. You can use registry 
commands to control how your datasets are published to registries, opting 
in or out on a dataset-by-dataset basis.

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

	// status represents the status command
	status := &cobra.Command{
		Use:   "status",
		Short: "get the status of a reference on the registry",
		Long: `
	use status to see what version of a dataset the registry has on-record, if any`,
		Example: `  Get status of a dataset reference::
		$ qri registry status me/dataset_name`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			// return o.Status()
			return fmt.Errorf("TODO (b5) = restore")
		},
	}

	// publish represents the publish command
	publish := &cobra.Command{
		Use:   "publish",
		Short: "Publish dataset info to the registry",
		Long: `
Publishes the dataset information onto the registry. There will be a record
of your dataset on the registry, and if your dataset is less than 20mbs, 
Qri will back your dataset up onto the registry.

Published datasets can be found by other peers using the ` + "`qri search`" + ` command.

Datasets are by default published to the registry when they are created.`,
		Example: `  Publish a dataset you've created to the registry:
  $ qri registry publish me/dataset_name`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			// return o.Publish()
			return fmt.Errorf("TODO (b5) = restore")
		},
	}

	// unpublish represents the unpublish command
	unpublish := &cobra.Command{
		Use:   "unpublish",
		Short: "remove dataset info from the registry",
		Long: `
Unpublish will remove the reference to your dataset from the registry. If 
you dataset was previously backed up onto the registry, this backup will 
be removed.

This dataset will no longer show up in search results.`,
		Example: `  Remove a dataset from the registry:
  $ qri registry unpublish me/dataset_name`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			// return o.Unpublish()
			return fmt.Errorf("TODO (b5) = restore")
		},
	}

	pin := &cobra.Command{
		Use:   "pin",
		Short: "pin dataset data to the registry",
		Long: `
Pin asks a registry to host a copy of your dataset, making it available for
others to download on the d.web`,
		Example: `  Pin a dataset to the registry:
  $ qri registry pin me/dataset_name`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			// return o.Pin()
			return fmt.Errorf("TODO (b5) = restore")
		},
	}

	unpin := &cobra.Command{
		Use:   "unpin",
		Short: "unpin dataset data from the registry",
		Long: `
Unpin reverses the pin process, asking a registry to remove it's hosted copy
of your dataset from the registry`,
		Example: `  Unpin a dataset from the registry:
  $ qri registry unpin me/dataset_name`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			// return o.Unpin()
			return fmt.Errorf("TODO (b5) = restore")
		},
	}

	cmd.AddCommand(publish, unpublish, status, pin, unpin)
	return cmd
}

// RegistryOptions encapsulates state for the registry command & subcommands
type RegistryOptions struct {
	ioes.IOStreams
	Refs                  []string
	RegistryClientMethods *lib.RegistryClientMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RegistryOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.RegistryClientMethods, err = f.RegistryClientMethods()
	return
}

// // Publish executes the publish command
// func (o *RegistryOptions) Publish() error {
// 	var res bool
// 	o.StartSpinner()
// 	defer o.StopSpinner()

// 	for _, arg := range o.Refs {
// 		ref, err := repo.ParseDatasetRef(arg)
// 		if err != nil {
// 			return err
// 		}

// 		if err = o.RegistryClientMethods.Publish(&ref, &res); err != nil {
// 			return err
// 		}
// 		printInfo(o.Out, "published dataset %s", ref)
// 	}
// 	return nil
// }

// // Status gets the status of a dataset reference on the registry
// func (o *RegistryOptions) Status() error {
// 	for _, arg := range o.Refs {
// 		o.StartSpinner()

// 		res := repo.DatasetRef{}

// 		ref, err := repo.ParseDatasetRef(arg)
// 		if err != nil {
// 			return err
// 		}

// 		err = o.RegistryClientMethods.GetDataset(&ref, &res)
// 		o.StopSpinner()
// 		if err != nil {
// 			printInfo(o.Out, "%s is not on this registry", ref.String())
// 		}

// 		if ref.Dataset != nil {
// 			refStr := refStringer(ref)
// 			fmt.Fprint(o.Out, refStr.String())
// 		}
// 	}

// 	return nil
// }

// // Unpublish executes the unpublish command
// func (o *RegistryOptions) Unpublish() error {
// 	var res bool
// 	o.StartSpinner()
// 	defer o.StopSpinner()

// 	for _, arg := range o.Refs {
// 		ref, err := repo.ParseDatasetRef(arg)
// 		if err != nil {
// 			return err
// 		}

// 		if err = o.RegistryClientMethods.Unpublish(&ref, &res); err != nil {
// 			return err
// 		}
// 		printInfo(o.Out, "unpublished dataset %s", ref)
// 	}
// 	return nil
// }

// // Pin executes the pin command
// func (o *RegistryOptions) Pin() error {
// 	var res bool
// 	o.StartSpinner()
// 	defer o.StopSpinner()

// 	for _, arg := range o.Refs {
// 		ref, err := repo.ParseDatasetRef(arg)
// 		if err != nil {
// 			return err
// 		}

// 		if err = o.RegistryClientMethods.Pin(&ref, &res); err != nil {
// 			return err
// 		}
// 		printInfo(o.Out, "pinned dataset %s", ref)
// 	}
// 	return nil
// }

// // Unpin executes the unpin command
// func (o *RegistryOptions) Unpin() error {
// 	var res bool
// 	o.StartSpinner()
// 	defer o.StopSpinner()

// 	for _, arg := range o.Refs {
// 		ref, err := repo.ParseDatasetRef(arg)
// 		if err != nil {
// 			return err
// 		}

// 		if err = o.RegistryClientMethods.Unpin(&ref, &res); err != nil {
// 			return err
// 		}
// 		printInfo(o.Out, "unpinned dataset %s", ref)
// 	}
// 	return nil
// }
