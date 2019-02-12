package cmd

import (
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// TODO: Tests.

// NewAddCommand creates an add command
func NewAddCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &AddOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a dataset from another peer",
		Annotations: map[string]string{
			"group": "dataset",
		},
		Long: `
Add retrieves a dataset owned by another peer and adds it to your repo. 
The dataset reference of the dataset will remain the same, including 
the name of the peer that originally added the dataset. You must have 
` + "`qri connect`" + ` running in another terminal to use this command.`,
		Example: `  add a dataset named their_data, owned by other_peer:
  $ qri add other_peer/their_data`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f); err != nil {
				return err
			}
			return o.Run(args)
		},
	}

	return cmd
}

// AddOptions encapsulates state for the add command
type AddOptions struct {
	ioes.IOStreams
	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *AddOptions) Complete(f Factory) error {
	q, err := f.Qri()
	if err != nil {
		return err
	}
	o.DatasetRequests = q.Datasets()
	return nil
}

// Run adds another peer's dataset to this user's repo
func (o *AddOptions) Run(args []string) error {
	o.StartSpinner()
	defer o.StopSpinner()

	for _, arg := range args {
		ref, err := parseCmdLineDatasetRef(arg)
		if err != nil {
			return err
		}

		res := repo.DatasetRef{}
		if err = o.DatasetRequests.Add(&ref, &res); err != nil {
			return err
		}

		printDatasetRefInfo(o.Out, 1, res)
		printInfo(o.Out, "Successfully added dataset %s", ref)
	}

	return nil
}
