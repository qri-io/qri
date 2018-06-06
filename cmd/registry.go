package cmd

import (
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// RegistryCmd is the subcommand for working with configured registries
var RegistryCmd = &cobra.Command{
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
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "publish dataset info to the registry",
	Example: `  Publish a dataset you've created to the registry:
	$ qri registry publish me/dataset_name`,
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req, err := registryRequests(false)
		ExitIfErr(err)

		var res bool
		for _, arg := range args {
			ref, err := repo.ParseDatasetRef(arg)
			ExitIfErr(err)

			err = req.Publish(&ref, &res)
			ExitIfErr(err)
			printInfo("published dataset %s", ref)
		}
	},
}

// unpublishCmd represents the unpublish command
var unpublishCmd = &cobra.Command{
	Use:   "unpublish",
	Short: "remove dataset info from the registry",
	Example: `  Remove a dataset from the registry:
  $ qri registry unpublish me/dataset_name`,
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req, err := registryRequests(false)
		ExitIfErr(err)

		var res bool
		for _, arg := range args {
			ref, err := repo.ParseDatasetRef(arg)
			ExitIfErr(err)

			err = req.Unpublish(&ref, &res)
			ExitIfErr(err)
			printInfo("unpublished dataset %s", ref)
		}
	},
}

func init() {
	RegistryCmd.AddCommand(publishCmd)
	RegistryCmd.AddCommand(unpublishCmd)
	RootCmd.AddCommand(RegistryCmd)
}
