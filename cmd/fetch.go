package cmd

import (
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewFetchCommand creates a `qri fetch` subcommand for working with configured registries
func NewFetchCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &FetchOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "fetch",
		Short:   "fetch logbook info for a dataset reference",
		Long:    ``,
		Example: ``,
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

	// cmd.Flags().BoolVarP(&o.Unfetch, "unfetch", "", false, "unfetch a dataset")
	// cmd.Flags().BoolVarP(&o.NoRegistry, "no-registry", "", false, "don't fetch to registry")
	// cmd.Flags().BoolVarP(&o.NoPin, "no-pin", "", false, "don't pin dataset to registry")
	cmd.Flags().StringVarP(&o.RemoteName, "remote", "", "", "name of remote to fetch to")

	return cmd
}

// FetchOptions encapsulates state for the fetch command
type FetchOptions struct {
	ioes.IOStreams

	Refs       []string
	Unfetch    bool
	NoRegistry bool
	NoPin      bool
	RemoteName string

	RemoteMethods *lib.RemoteMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *FetchOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.RemoteMethods, err = f.RemoteMethods()
	return
}

// Run executes the fetch command
func (o *FetchOptions) Run() error {
	var res repo.DatasetRef
	for _, ref := range o.Refs {
		p := lib.FetchParams{
			Ref:        ref,
			RemoteName: o.RemoteName,
		}
		if err := o.RemoteMethods.Fetch(&p, &res); err != nil {
			return err
		}
		printInfo(o.Out, "fetched dataset %s", res)
	}
	return nil
}
